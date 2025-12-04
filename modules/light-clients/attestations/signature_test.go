package attestations_test

import (
	"bytes"
	"crypto/sha256"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	attestations "github.com/cosmos/ibc-go/v10/modules/light-clients/attestations"
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
				hash := sha256.Sum256(attestationData)
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
				hash := sha256.Sum256(attestationData)
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

	hash := sha256.Sum256(attestationData)
	sig, err := crypto.Sign(hash[:], privKey)
	s.Require().NoError(err)

	proof := &attestations.AttestationProof{
		AttestationData: attestationData,
		Signatures:      [][]byte{sig},
	}

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)
}
