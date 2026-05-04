package types_test

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/ibc-go/prototypes/x/ift/types"
	solanapb "github.com/cosmos/solidity-ibc-eureka/packages/go-proto/solana"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	gmptypes "github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
)

const (
	testCounterpartyClientID = "08-wasm-0" // Solana client tracking Cosmos
	testSenderAddress        = "wf1ift"
)

// TestConstructMintCall_CosmosTx_Deserializable verifies that the payload
// constructed by ConstructMintCall can be deserialized by the GMP module.
// This test catches the bug where JSON encoding with @type field was used
// instead of proper protobuf CosmosTx encoding.
func TestConstructMintCall_CosmosTx_Deserializable(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	registry.RegisterImplementations((*sdk.Msg)(nil), &types.MsgIFTMint{})
	cdc := codec.NewProtoCodec(registry)

	receiver := "cosmos1y6xz2ggfc0pcsmyjlekh0j9pxh6hk87yfwcjct"
	icaAddress := "cosmos1uu635yk0hz3cvrypnryrggltrjq7975jrmeg97"
	amount := math.NewInt(1000000)
	denom := "testdenom"

	payload, err := types.ConstructMintCall(
		cdc,
		receiver,
		amount,
		types.ConstructorCosmos,
		denom,
		icaAddress,
	)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	// Verify payload can be deserialized by GMP module
	msgs, err := gmptypes.DeserializeCosmosTx(cdc, payload)
	require.NoError(t, err, "GMP module should be able to deserialize the payload")
	require.Len(t, msgs, 1)

	mintMsg, ok := msgs[0].(*types.MsgIFTMint)
	require.True(t, ok, "deserialized message should be MsgIFTMint")
	require.Equal(t, receiver, mintMsg.Receiver)
	require.Equal(t, icaAddress, mintMsg.Signer)
	require.Equal(t, denom, mintMsg.Denom)
	require.True(t, amount.Equal(mintMsg.Amount))
}

// TestConstructMintCall_UnknownConstructor verifies that unknown constructor
// types return an error.
func TestConstructMintCall_UnknownConstructor(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	registry.RegisterImplementations((*sdk.Msg)(nil), &types.MsgIFTMint{})
	cdc := codec.NewProtoCodec(registry)

	_, err := types.ConstructMintCall(
		cdc,
		"cosmos1y6xz2ggfc0pcsmyjlekh0j9pxh6hk87yfwcjct",
		math.NewInt(1000),
		"invalid",
		"testdenom",
		"cosmos1uu635yk0hz3cvrypnryrggltrjq7975jrmeg97",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid constructor type")
}

// TestConstructMintCall_EVM_ValidAddress verifies valid EVM addresses are accepted
func TestConstructMintCall_EVM_ValidAddress(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	validAddresses := []string{
		"0x1234567890abcdef1234567890abcdef12345678",
		"0xABCDEF1234567890ABCDEF1234567890ABCDEF12",
		"0x1111111111111111111111111111111111111111",
	}

	for _, addr := range validAddresses {
		payload, err := types.ConstructMintCall(
			cdc,
			addr,
			math.NewInt(1000),
			types.ConstructorEVM,
			"", "",
		)
		require.NoError(t, err, "valid address %s should be accepted", addr)
		require.NotEmpty(t, payload)
		expectedSelector := crypto.Keccak256([]byte("iftMint(address,uint256)"))[:4]
		require.Equal(t, expectedSelector, payload[:4], "should have correct selector")
	}
}

// TestConstructMintCall_EVM_InvalidAddress verifies invalid EVM addresses are rejected
func TestConstructMintCall_EVM_InvalidAddress(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	cases := []struct {
		name     string
		receiver string
		errMsg   string
	}{
		{
			name:     "cosmos address",
			receiver: "cosmos1y6xz2ggfc0pcsmyjlekh0j9pxh6hk87yfwcjct",
			errMsg:   "invalid EVM address",
		},
		{
			name:     "invalid hex characters",
			receiver: "0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG",
			errMsg:   "invalid EVM address",
		},
		{
			name:     "too short",
			receiver: "0x1234",
			errMsg:   "invalid EVM address",
		},
		{
			name:     "empty string",
			receiver: "",
			errMsg:   "invalid EVM address",
		},
		{
			name:     "plain text",
			receiver: "invalid",
			errMsg:   "invalid EVM address",
		},
		{
			name:     "too long",
			receiver: "0x1234567890abcdef1234567890abcdef1234567890",
			errMsg:   "invalid EVM address",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := types.ConstructMintCall(
				cdc,
				tc.receiver,
				math.NewInt(1000),
				types.ConstructorEVM,
				"", "",
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

// TestConstructMintCall_EVM_ZeroAddress verifies zero address is rejected
func TestConstructMintCall_EVM_ZeroAddress(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	_, err := types.ConstructMintCall(
		cdc,
		"0x0000000000000000000000000000000000000000",
		math.NewInt(1000),
		types.ConstructorEVM,
		"", "",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "zero address not allowed")
}

// TestConstructMintCall_Cosmos_ValidAddresses verifies valid Cosmos addresses with different prefixes are accepted
func TestConstructMintCall_Cosmos_ValidAddresses(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	registry.RegisterImplementations((*sdk.Msg)(nil), &types.MsgIFTMint{})
	cdc := codec.NewProtoCodec(registry)

	// Valid bech32 addresses with different prefixes
	validAddresses := []string{
		"cosmos1y6xz2ggfc0pcsmyjlekh0j9pxh6hk87yfwcjct",
		"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02",
	}

	for _, addr := range validAddresses {
		payload, err := types.ConstructMintCall(
			cdc,
			addr,
			math.NewInt(1000),
			types.ConstructorCosmos,
			"testdenom",
			"cosmos1uu635yk0hz3cvrypnryrggltrjq7975jrmeg97",
		)
		require.NoError(t, err, "valid address %s should be accepted", addr)
		require.NotEmpty(t, payload)
	}
}

// TestConstructMintCall_Cosmos_InvalidAddresses verifies invalid Cosmos addresses are rejected
func TestConstructMintCall_Cosmos_InvalidAddresses(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	registry.RegisterImplementations((*sdk.Msg)(nil), &types.MsgIFTMint{})
	cdc := codec.NewProtoCodec(registry)

	cases := []struct {
		name     string
		receiver string
		errMsg   string
	}{
		{
			name:     "empty string",
			receiver: "",
			errMsg:   "address cannot be empty",
		},
		{
			name:     "whitespace only",
			receiver: "   ",
			errMsg:   "address cannot be empty",
		},
		{
			name:     "invalid bech32",
			receiver: "notabech32address",
			errMsg:   "invalid bech32",
		},
		{
			name:     "EVM address",
			receiver: "0x742d35Cc6634C0532925a3b844Bc9e7595f3aD12",
			errMsg:   "invalid bech32",
		},
		{
			name:     "truncated address",
			receiver: "cosmos1abc",
			errMsg:   "invalid bech32",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := types.ConstructMintCall(
				cdc,
				tc.receiver,
				math.NewInt(1000),
				types.ConstructorCosmos,
				"testdenom",
				"cosmos1uu635yk0hz3cvrypnryrggltrjq7975jrmeg97",
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

// TestSolanaConstructor_ConstructMintCall_ValidAddress verifies valid Solana addresses are accepted
func TestSolanaConstructor_ConstructMintCall_ValidAddress(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	// Setup constructor with test config using well-known program IDs
	constructor := &types.SolanaConstructor{
		IFTProgramID:         solana.TokenProgramID,
		GMPProgramID:         solana.Token2022ProgramID,
		Mint:                 solana.SPLAssociatedTokenAccountProgramID,
		SenderAddress:        testSenderAddress,
		CounterpartyClientID: testCounterpartyClientID,
	}

	validAddresses := []string{
		"9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin",
		solana.SystemProgramID.String(),
		solana.TokenProgramID.String(),
	}

	for _, addr := range validAddresses {
		payload, err := constructor.ConstructMintCall(
			cdc,
			addr,
			math.NewInt(1000),
			"", "",
		)
		require.NoError(t, err, "valid address %s should be accepted", addr)
		require.NotEmpty(t, payload)

		// Verify payload is valid GMPSolanaPayload
		var solanaPayload solanapb.GMPSolanaPayload
		err = proto.Unmarshal(payload, &solanaPayload)
		require.NoError(t, err, "payload should be valid GMPSolanaPayload")
		require.NotEmpty(t, solanaPayload.Accounts)
		require.NotEmpty(t, solanaPayload.Data)
		require.Equal(t, uint64(3_000_000), solanaPayload.PrefundLamports)
	}
}

// TestSolanaConstructor_ConstructMintCall_InvalidAddress verifies invalid Solana addresses are rejected
func TestSolanaConstructor_ConstructMintCall_InvalidAddress(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	constructor := &types.SolanaConstructor{
		IFTProgramID:         solana.TokenProgramID,
		GMPProgramID:         solana.Token2022ProgramID,
		Mint:                 solana.SPLAssociatedTokenAccountProgramID,
		SenderAddress:        testSenderAddress,
		CounterpartyClientID: testCounterpartyClientID,
	}

	cases := []struct {
		name     string
		receiver string
		errMsg   string
	}{
		{
			name:     "cosmos address",
			receiver: "cosmos1y6xz2ggfc0pcsmyjlekh0j9pxh6hk87yfwcjct",
			errMsg:   "invalid Solana address",
		},
		{
			name:     "EVM address",
			receiver: "0x742d35Cc6634C0532925a3b844Bc9e7595f3aD12",
			errMsg:   "invalid Solana address",
		},
		{
			name:     "empty string",
			receiver: "",
			errMsg:   "invalid Solana address",
		},
		{
			name:     "invalid base58",
			receiver: "0OIl11111111111111111111111111111",
			errMsg:   "invalid Solana address",
		},
		{
			name:     "too short",
			receiver: "abc123",
			errMsg:   "invalid Solana address",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := constructor.ConstructMintCall(
				cdc,
				tc.receiver,
				math.NewInt(1000),
				"", "",
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

// TestSolanaConstructor_ConstructMintCall_PayloadStructure verifies the payload structure
func TestSolanaConstructor_ConstructMintCall_PayloadStructure(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	constructor := &types.SolanaConstructor{
		IFTProgramID:         solana.TokenProgramID,
		GMPProgramID:         solana.Token2022ProgramID,
		Mint:                 solana.SPLAssociatedTokenAccountProgramID,
		SenderAddress:        testSenderAddress,
		CounterpartyClientID: testCounterpartyClientID,
	}

	receiver := "9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin"
	amount := math.NewInt(1000000)

	payload, err := constructor.ConstructMintCall(cdc, receiver, amount, "", "")
	require.NoError(t, err)

	var solanaPayload solanapb.GMPSolanaPayload
	err = proto.Unmarshal(payload, &solanaPayload)
	require.NoError(t, err)

	// Verify accounts list (12 accounts, payer is explicit GMP PDA at position 8)
	require.Len(t, solanaPayload.Accounts, 12, "should have 12 accounts (payer is explicit)")

	require.Equal(t, uint64(3_000_000), solanaPayload.PrefundLamports)

	expectedDiscriminator := sha256.Sum256([]byte("global:ift_mint"))
	require.Equal(t, expectedDiscriminator[:8], solanaPayload.Data[:8], "should have correct Anchor discriminator")

	data := solanaPayload.Data[8:] // skip discriminator

	receiverPubkey, err := solana.PublicKeyFromBase58(receiver)
	require.NoError(t, err)
	require.Equal(t, receiverPubkey[:], data[:32], "receiver pubkey should match")

	decodedAmount := binary.LittleEndian.Uint64(data[32:40])
	require.Equal(t, amount.Uint64(), decodedAmount, "amount should match")
}

// TestSolanaConstructor_ConstructMintCall_MissingConfig verifies error when Solana config is missing
func TestSolanaConstructor_ConstructMintCall_MissingConfig(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	constructor := &types.SolanaConstructor{} // Missing Solana config (zero values)

	_, err := constructor.ConstructMintCall(
		cdc,
		"9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin",
		math.NewInt(1000),
		"", "",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "solana config is required")
}
