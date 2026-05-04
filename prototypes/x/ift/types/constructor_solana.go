package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/cosmos/gogoproto/proto"
	solanapb "github.com/cosmos/solidity-ibc-eureka/packages/go-proto/solana"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
)

// IFTMintMsg is the Borsh-encoded message for the ift_mint instruction
type IFTMintMsg struct {
	Receiver solana.PublicKey
	Amount   uint64
}

// SolanaConstructor handles mint call construction for Solana.
// It holds the Solana-specific configuration needed for mint calls.
type SolanaConstructor struct {
	IFTProgramID         solana.PublicKey
	GMPProgramID         solana.PublicKey
	Mint                 solana.PublicKey
	SenderAddress        string
	CounterpartyClientID string
}

// NewSolanaConstructor creates a fully configured SolanaConstructor from a constructor string
// and additional context needed for mint calls.
func NewSolanaConstructor(constructorStr, counterpartyIftAddress, senderAddress, counterpartyClientID string) (*SolanaConstructor, error) {
	cfg, err := ParseSolanaConfig(constructorStr)
	if err != nil {
		return nil, err
	}

	iftProgramID, err := solana.PublicKeyFromBase58(counterpartyIftAddress)
	if err != nil {
		return nil, ErrInvalidSolanaAddress.Wrapf("invalid IFT program ID: %s", err)
	}

	gmpProgramID, err := solana.PublicKeyFromBase58(cfg.GMPProgramID)
	if err != nil {
		return nil, ErrInvalidSolanaAddress.Wrapf("invalid GMP program ID: %s", err)
	}

	mint, err := solana.PublicKeyFromBase58(cfg.MintPubKey)
	if err != nil {
		return nil, ErrInvalidSolanaAddress.Wrapf("invalid mint pubkey: %s", err)
	}

	return &SolanaConstructor{
		IFTProgramID:         iftProgramID,
		GMPProgramID:         gmpProgramID,
		Mint:                 mint,
		SenderAddress:        senderAddress,
		CounterpartyClientID: counterpartyClientID,
	}, nil
}

func (SolanaConstructor) ValidateCounterpartyAddress(address string) error {
	return ValidateSolanaAddress(address)
}

// ValidateSolanaAddress validates that an address is a valid base58-encoded Solana public key
func ValidateSolanaAddress(address string) error {
	if address == "" {
		return ErrInvalidSolanaAddress.Wrap("address cannot be empty")
	}

	_, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return ErrInvalidSolanaAddress.Wrapf("invalid base58 pubkey: %s", err)
	}

	return nil
}

// ConstructMintCall builds the GMPSolanaPayload for a Solana IFT mint instruction.
func (s SolanaConstructor) ConstructMintCall(_ codec.BinaryCodec, receiver string, amount math.Int, _, _ string) ([]byte, error) {
	if s.IFTProgramID.IsZero() {
		return nil, ErrConstructMintCallFailed.Wrap("solana config is required")
	}

	if err := ValidateSolanaAddress(receiver); err != nil {
		return nil, err
	}

	receiverPubkey, _ := solana.PublicKeyFromBase58(receiver)

	// Derive all PDAs
	appStatePDA := deriveAppStatePDA(s.IFTProgramID)
	appMintStatePDA := deriveAppMintStatePDA(s.IFTProgramID, s.Mint)
	iftBridgePDA := deriveIFTBridgePDA(s.IFTProgramID, s.Mint, s.CounterpartyClientID)
	mintAuthorityPDA := deriveMintAuthorityPDA(s.IFTProgramID, s.Mint)
	receiverATA := deriveAssociatedTokenAddress(s.Mint, receiverPubkey)
	gmpAccountPDA, _ := deriveGMPAccountPDA(s.GMPProgramID, s.CounterpartyClientID, s.SenderAddress)

	// Solana programs only accept uint64 amounts; reject anything that wouldn't
	// round-trip cleanly rather than silently truncating via .Uint64().
	if !amount.BigInt().IsUint64() {
		return nil, ErrConstructMintCallFailed.Wrapf("amount %s overflows uint64", amount)
	}

	// Build instruction data: discriminator + borsh(IFTMintMsg)
	data, err := buildInstructionData(receiverPubkey, amount.BigInt().Uint64())
	if err != nil {
		return nil, ErrConstructMintCallFailed.Wrapf("failed to build instruction data: %s", err)
	}

	// Build accounts list matching IFTMint struct order.
	// The GMP PDA acts as payer (pre-funded by relayer) for ATA creation.
	accounts := []*solanapb.SolanaAccountMeta{
		{Pubkey: appStatePDA[:], IsSigner: false, IsWritable: false},                               // 0: app_state
		{Pubkey: appMintStatePDA[:], IsSigner: false, IsWritable: true},                            // 1: app_mint_state
		{Pubkey: iftBridgePDA[:], IsSigner: false, IsWritable: false},                              // 2: ift_bridge
		{Pubkey: s.Mint[:], IsSigner: false, IsWritable: true},                                     // 3: mint
		{Pubkey: mintAuthorityPDA[:], IsSigner: false, IsWritable: false},                          // 4: mint_authority
		{Pubkey: receiverATA[:], IsSigner: false, IsWritable: true},                                // 5: receiver_token_account
		{Pubkey: receiverPubkey[:], IsSigner: false, IsWritable: false},                            // 6: receiver_owner
		{Pubkey: gmpAccountPDA[:], IsSigner: true, IsWritable: false},                              // 7: gmp_account (signer via CPI)
		{Pubkey: gmpAccountPDA[:], IsSigner: true, IsWritable: true},                               // 8: payer (GMP PDA, pre-funded by relayer)
		{Pubkey: solana.TokenProgramID[:], IsSigner: false, IsWritable: false},                     // 9: token_program
		{Pubkey: solana.SPLAssociatedTokenAccountProgramID[:], IsSigner: false, IsWritable: false}, // 10: associated_token_program
		{Pubkey: solana.SystemProgramID[:], IsSigner: false, IsWritable: false},                    // 11: system_program
	}

	// Build GMPSolanaPayload
	// prefund_lamports covers ATA creation rent (~2.04M) + GMP PDA rent-exempt minimum (~890K)
	payload := &solanapb.GMPSolanaPayload{
		Accounts:        accounts,
		Data:            data,
		PrefundLamports: 3_000_000,
	}

	// NOTE: Use of protobuf is for consistency and might require removal as decryption is less from optimal
	// and will influence increase of CU in case of supporting multiple packets
	return proto.Marshal(payload)
}

// buildInstructionData builds the Anchor instruction data: discriminator + borsh(IFTMintMsg)
func buildInstructionData(receiver solana.PublicKey, amount uint64) ([]byte, error) {
	// Anchor discriminator: SHA256("global:ift_mint")[0:8]
	discriminatorHash := sha256.Sum256([]byte("global:ift_mint"))
	discriminator := discriminatorHash[:8]

	// Borsh-encode IFTMintMsg
	msg := IFTMintMsg{
		Receiver: receiver,
		Amount:   amount,
	}

	var buf bytes.Buffer
	encoder := bin.NewBorshEncoder(&buf)
	if err := encoder.Encode(msg); err != nil {
		return nil, err
	}

	// Combine discriminator + borsh-encoded msg
	return append(discriminator, buf.Bytes()...), nil
}

// deriveAppStatePDA derives the IFT app_state PDA: ["ift_app_state"]
func deriveAppStatePDA(iftProgramID solana.PublicKey) solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("ift_app_state"),
		},
		iftProgramID,
	)
	return pda
}

// deriveAppMintStatePDA derives the IFT app_mint_state PDA: ["ift_app_mint_state", mint]
func deriveAppMintStatePDA(iftProgramID, mint solana.PublicKey) solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("ift_app_mint_state"),
			mint[:],
		},
		iftProgramID,
	)
	return pda
}

// deriveIFTBridgePDA derives the IFT bridge PDA: ["ift_bridge", mint, client_id]
func deriveIFTBridgePDA(iftProgramID, mint solana.PublicKey, clientID string) solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("ift_bridge"),
			mint[:],
			[]byte(clientID),
		},
		iftProgramID,
	)
	return pda
}

// deriveMintAuthorityPDA derives the mint authority PDA: ["ift_mint_authority", mint]
func deriveMintAuthorityPDA(iftProgramID, mint solana.PublicKey) solana.PublicKey {
	pda, _, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("ift_mint_authority"),
			mint[:],
		},
		iftProgramID,
	)
	return pda
}

// deriveAssociatedTokenAddress derives the ATA for a given owner and mint
func deriveAssociatedTokenAddress(mint, owner solana.PublicKey) solana.PublicKey {
	ata, _, _ := solana.FindProgramAddress(
		[][]byte{
			owner[:],
			solana.TokenProgramID[:],
			mint[:],
		},
		solana.SPLAssociatedTokenAccountProgramID,
	)
	return ata
}

// deriveGMPAccountPDA derives the GMP account PDA and bump.
// Seeds: ["gmp_account", account_identifier_digest]
// Where account_identifier_digest = SHA256(borsh(client_id, sender_address, salt))
func deriveGMPAccountPDA(gmpProgramID solana.PublicKey, clientID, senderAddress string) (solana.PublicKey, uint8) {
	// Borsh serialization: u32_le(len) + bytes for each string field
	var buf bytes.Buffer

	clientBytes := []byte(clientID)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(clientBytes)))
	_, _ = buf.Write(clientBytes)

	senderBytes := []byte(senderAddress)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(senderBytes)))
	_, _ = buf.Write(senderBytes)

	_ = binary.Write(&buf, binary.LittleEndian, uint32(0)) // empty salt

	identifierHash := sha256.Sum256(buf.Bytes())

	pda, bump, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("gmp_account"),
			identifierHash[:],
		},
		gmpProgramID,
	)
	return pda, bump
}
