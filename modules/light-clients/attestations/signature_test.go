package attestations_test

import (
	"bytes"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	"github.com/cosmos/ibc-go/v10/modules/light-clients/attestations"
)

func (s *AttestationsTestSuite) TestVerifySignatures() {
	testCases := []struct {
		name        string
		setupProof  func(attestationData []byte) *attestations.AttestationProof
		errContains string
	}{
		{
			name: "failure: invalid signature length",
			setupProof: func(attestationData []byte) *attestations.AttestationProof {
				return &attestations.AttestationProof{
					AttestationData: attestationData,
					Signatures:      [][]byte{bytes.Repeat([]byte{0x01}, 32)},
				}
			},
			errContains: "invalid length",
		},
		{
			name: "failure: unknown signer",
			setupProof: func(attestationData []byte) *attestations.AttestationProof {
				unknownKey, _ := crypto.GenerateKey()
				hash := attestations.TaggedSigningInput(attestationData, attestations.AttestationTypeState)
				unknownSig, _ := crypto.Sign(hash[:], unknownKey)
				return &attestations.AttestationProof{
					AttestationData: attestationData,
					Signatures:      [][]byte{unknownSig},
				}
			},
			errContains: "not in attestor set",
		},
		{
			name: "failure: duplicate signer",
			setupProof: func(attestationData []byte) *attestations.AttestationProof {
				hash := attestations.TaggedSigningInput(attestationData, attestations.AttestationTypeState)
				sig1, _ := crypto.Sign(hash[:], s.attestors[0])
				sig2, _ := crypto.Sign(hash[:], s.attestors[0])
				return &attestations.AttestationProof{
					AttestationData: attestationData,
					Signatures:      [][]byte{sig1, sig2},
				}
			},
			errContains: "duplicate signer",
		},
		{
			name: "failure: empty signatures",
			setupProof: func(attestationData []byte) *attestations.AttestationProof {
				return &attestations.AttestationProof{
					AttestationData: attestationData,
					Signatures:      [][]byte{},
				}
			},
			errContains: "signatures cannot be empty",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			initialHeight := uint64(100)
			initialTimestamp := uint64(time.Second.Nanoseconds())
			clientID := testClientID
			ctx := s.chainA.GetContext()

			s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)

			newHeight := uint64(200)
			newTimestamp := uint64(2 * time.Second.Nanoseconds())
			attestationData := s.createStateAttestation(newHeight, newTimestamp)

			proof := tc.setupProof(attestationData)

			err := s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
			s.Require().Error(err)
			s.Require().ErrorContains(err, tc.errContains)
		})
	}
}

func (s *AttestationsTestSuite) TestAddressCaseInsensitiveComparison() {
	privKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	addr := crypto.PubkeyToAddress(privKey.PublicKey).Hex()

	lowercaseAddrs := []string{strings.ToLower(addr)}
	clientState := attestations.NewClientState(lowercaseAddrs, 1, 100)
	consensusState := s.createConsensusState(uint64(time.Second.Nanoseconds()))

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := "attestations-case-test"
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	hash := attestations.TaggedSigningInput(attestationData, attestations.AttestationTypeState)
	sig, err := crypto.Sign(hash[:], privKey)
	s.Require().NoError(err)

	proof := &attestations.AttestationProof{
		AttestationData: attestationData,
		Signatures:      [][]byte{sig},
	}

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)
}

func (s *AttestationsTestSuite) TestCrossDomainReplay() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())
	clientID := testClientID
	ctx := s.chainA.GetContext()

	s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	// Sign as Packet, verify as State — must fail
	packetProof := s.createAttestationProof(attestationData, []int{0, 1, 2}, attestations.AttestationTypePacket)
	err := s.lightClientModule.VerifyClientMessage(ctx, clientID, packetProof)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "not in attestor set")

	// Sign as State, verify as Packet (via membership) — must fail
	hashedPath := crypto.Keccak256(bytes.Repeat([]byte{0x01}, 32))
	value := bytes.Repeat([]byte{0xAB}, 32)
	packetAttestation := s.createPacketAttestation(initialHeight, []attestations.PacketCompact{{Path: hashedPath, Commitment: value}})
	stateSignedProof := s.createAttestationProof(packetAttestation, []int{0, 1, 2}, attestations.AttestationTypeState)
	proofBz := s.marshalProof(stateSignedProof)

	proofHeight := clienttypes.NewHeight(0, initialHeight)
	path := commitmenttypesv2.NewMerklePath(bytes.Repeat([]byte{0x01}, 32))
	err = s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, proofBz, path, value)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "not in attestor set")

	// Sign as State, verify as Packet (via non-membership) — must fail
	zeroCommitment := make([]byte, 32)
	nonMemPath := bytes.Repeat([]byte{0x02}, 32)
	hashedNonMemPath := crypto.Keccak256(nonMemPath)
	nonMemPacketAttestation := s.createPacketAttestation(initialHeight, []attestations.PacketCompact{{Path: hashedNonMemPath, Commitment: zeroCommitment}})
	stateSignedNonMemProof := s.createAttestationProof(nonMemPacketAttestation, []int{0, 1, 2}, attestations.AttestationTypeState)
	nonMemProofBz := s.marshalProof(stateSignedNonMemProof)

	err = s.lightClientModule.VerifyNonMembership(ctx, clientID, proofHeight, 0, 0, nonMemProofBz, commitmenttypesv2.NewMerklePath(nonMemPath))
	s.Require().Error(err)
	s.Require().ErrorContains(err, "not in attestor set")
}
