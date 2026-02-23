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

func (s *AttestationsTestSuite) TestClientStateValidate() {
	testCases := []struct {
		name        string
		clientState *attestations.ClientState
		expErr      bool
	}{
		{
			name:        "valid client state",
			clientState: s.createClientState(1),
			expErr:      false,
		},
		{
			name:        "zero latest height",
			clientState: attestations.NewClientState(s.attestorAddrs, s.minRequiredSigs, 0),
			expErr:      true,
		},
		{
			name:        "empty attestor addresses",
			clientState: attestations.NewClientState([]string{}, 1, 1),
			expErr:      true,
		},
		{
			name:        "zero min required sigs",
			clientState: attestations.NewClientState(s.attestorAddrs, 0, 1),
			expErr:      true,
		},
		{
			name:        "min required sigs exceeds attestor count",
			clientState: attestations.NewClientState(s.attestorAddrs, 10, 1),
			expErr:      true,
		},
		{
			name:        "duplicate attestor address",
			clientState: attestations.NewClientState([]string{s.attestorAddrs[0], s.attestorAddrs[0]}, 1, 1),
			expErr:      true,
		},
		{
			name:        "empty attestor address",
			clientState: attestations.NewClientState([]string{""}, 1, 1),
			expErr:      true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.clientState.Validate()
			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *AttestationsTestSuite) TestClientStateValidateInvalidAddressFormat() {
	testCases := []struct {
		name        string
		clientState *attestations.ClientState
		expErr      string
	}{
		{
			name:        "invalid address format - not hex",
			clientState: attestations.NewClientState([]string{"not-a-valid-address"}, 1, 1),
			expErr:      "invalid attestor address format",
		},
		{
			name:        "invalid address format - too short",
			clientState: attestations.NewClientState([]string{"0x1234"}, 1, 1),
			expErr:      "invalid attestor address format",
		},
		{
			name:        "valid checksummed address",
			clientState: attestations.NewClientState([]string{s.attestorAddrs[0]}, 1, 1),
			expErr:      "",
		},
		{
			name:        "valid lowercase address",
			clientState: attestations.NewClientState([]string{strings.ToLower(s.attestorAddrs[0])}, 1, 1),
			expErr:      "",
		},
		{
			name:        "duplicate addresses with different case",
			clientState: attestations.NewClientState([]string{s.attestorAddrs[0], strings.ToLower(s.attestorAddrs[0])}, 1, 1),
			expErr:      "duplicate attestor address",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.clientState.Validate()
			if tc.expErr != "" {
				s.Require().ErrorContains(err, tc.expErr)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *AttestationsTestSuite) TestVerifyMembership() {
	testCases := []struct {
		name         string
		freezeClient bool
		pathToVerify []byte
		attestedPath []byte
		value        []byte
		expErr       error
	}{
		{
			name:         "success: matching path and commitment",
			freezeClient: false,
			pathToVerify: bytes.Repeat([]byte{0x01}, 32),
			attestedPath: bytes.Repeat([]byte{0x01}, 32),
			value:        bytes.Repeat([]byte{0xAB}, 32),
			expErr:       nil,
		},
		{
			name:         "failure: mismatched path",
			freezeClient: false,
			pathToVerify: bytes.Repeat([]byte{0x02}, 32),
			attestedPath: bytes.Repeat([]byte{0x01}, 32),
			value:        bytes.Repeat([]byte{0xCD}, 32),
			expErr:       attestations.ErrNotMember,
		},
		{
			name:         "failure: frozen client",
			freezeClient: true,
			pathToVerify: bytes.Repeat([]byte{0x01}, 32),
			attestedPath: bytes.Repeat([]byte{0x01}, 32),
			value:        bytes.Repeat([]byte{0xAB}, 32),
			expErr:       attestations.ErrClientFrozen,
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
			stateAttestation := s.createStateAttestation(newHeight, newTimestamp)
			signers := []int{0, 1, 2}
			updateProof := s.createAttestationProof(stateAttestation, signers, attestations.AttestationTypeState)

			err := s.lightClientModule.VerifyClientMessage(ctx, clientID, updateProof)
			s.Require().NoError(err)

			heights := s.lightClientModule.UpdateState(ctx, clientID, updateProof)
			s.Require().Len(heights, 1)

			if tc.freezeClient {
				s.freezeClient(ctx, clientID)
			}

			// Commitment is stored as raw value (not hashed)
			hashedPath := crypto.Keccak256(tc.attestedPath)
			packetAttestation := s.createPacketAttestation(newHeight, []attestations.PacketCompact{{Path: hashedPath, Commitment: tc.value}})
			membershipProof := s.createAttestationProof(packetAttestation, signers, attestations.AttestationTypePacket)
			membershipProofBz := s.marshalProof(membershipProof)

			proofHeight := clienttypes.NewHeight(0, newHeight)
			err = s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, membershipProofBz, commitmenttypesv2.NewMerklePath(tc.pathToVerify), tc.value)

			if tc.expErr != nil {
				s.Require().ErrorIs(err, tc.expErr)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *AttestationsTestSuite) TestVerifyMembershipMalformedProof() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())
	clientID := testClientID
	ctx := s.chainA.GetContext()

	s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)

	proofHeight := clienttypes.NewHeight(0, initialHeight)
	path := commitmenttypesv2.NewMerklePath(bytes.Repeat([]byte{0x01}, 32))
	value := bytes.Repeat([]byte{0xAB}, 32)

	malformedProof := []byte("invalid-protobuf-data")
	err := s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, malformedProof, path, value)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "failed to unmarshal proof")

	malformedData := []byte("invalid-packet-attestation-data")
	proof := s.createAttestationProof(malformedData, []int{0, 1, 2}, attestations.AttestationTypePacket)
	proofBz := s.marshalProof(proof)

	err = s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, proofBz, path, value)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "failed to ABI decode attestation data")
}

func (s *AttestationsTestSuite) TestVerifyMembershipVariableLengthPath() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())
	clientID := testClientID
	ctx := s.chainA.GetContext()

	s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	s.updateClientState(ctx, clientID, newHeight, newTimestamp)

	shortPath := []byte("attestations-0\x01\x00\x00\x00\x00\x00\x00\x00\x01")
	path := commitmenttypesv2.NewMerklePath(shortPath)
	value32 := bytes.Repeat([]byte{0xAB}, 32)

	hashedPath := crypto.Keccak256(shortPath)
	packetAttestation := s.createPacketAttestation(newHeight, []attestations.PacketCompact{{Path: hashedPath, Commitment: value32}})
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(packetAttestation, signers, attestations.AttestationTypePacket)
	proofBz := s.marshalProof(proof)

	proofHeight := clienttypes.NewHeight(0, newHeight)
	err := s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, proofBz, path, value32)
	s.Require().NoError(err)
}

func (s *AttestationsTestSuite) TestVerifyMembershipInvalidKeyPathLength() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())
	clientID := testClientID
	ctx := s.chainA.GetContext()

	s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	s.updateClientState(ctx, clientID, newHeight, newTimestamp)

	value32 := bytes.Repeat([]byte{0xAB}, 32)
	hashedPath := crypto.Keccak256(bytes.Repeat([]byte{0x01}, 32))
	packetAttestation := s.createPacketAttestation(newHeight, []attestations.PacketCompact{{Path: hashedPath, Commitment: value32}})
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(packetAttestation, signers, attestations.AttestationTypePacket)
	proofBz := s.marshalProof(proof)
	proofHeight := clienttypes.NewHeight(0, newHeight)

	emptyPath := commitmenttypesv2.MerklePath{KeyPath: [][]byte{}}
	err := s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, proofBz, emptyPath, value32)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "path cannot be empty")

	multiKeyPath := commitmenttypesv2.NewMerklePath(bytes.Repeat([]byte{0x01}, 32), bytes.Repeat([]byte{0x02}, 32))
	err = s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, proofBz, multiKeyPath, value32)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "key path must have exactly 1 element, got 2")
}

func (s *AttestationsTestSuite) TestVerifyNonMembershipInvalidKeyPathLength() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())
	clientID := testClientID
	ctx := s.chainA.GetContext()

	s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	s.updateClientState(ctx, clientID, newHeight, newTimestamp)

	hashedPath := crypto.Keccak256(bytes.Repeat([]byte{0x01}, 32))
	packetAttestation := s.createPacketAttestation(newHeight, []attestations.PacketCompact{{Path: hashedPath, Commitment: make([]byte, 32)}})
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(packetAttestation, signers, attestations.AttestationTypePacket)
	proofBz := s.marshalProof(proof)
	proofHeight := clienttypes.NewHeight(0, newHeight)

	emptyPath := commitmenttypesv2.MerklePath{KeyPath: [][]byte{}}
	err := s.lightClientModule.VerifyNonMembership(ctx, clientID, proofHeight, 0, 0, proofBz, emptyPath)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "path cannot be empty")

	multiKeyPath := commitmenttypesv2.NewMerklePath(bytes.Repeat([]byte{0x01}, 32), bytes.Repeat([]byte{0x02}, 32))
	err = s.lightClientModule.VerifyNonMembership(ctx, clientID, proofHeight, 0, 0, proofBz, multiKeyPath)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "key path must have exactly 1 element, got 2")
}

func (s *AttestationsTestSuite) TestVerifyNonMembership() {
	testCases := []struct {
		name         string
		freezeClient bool
		pathToVerify []byte
		packets      []attestations.PacketCompact
		expErr       error
	}{
		{
			name:         "success: zero commitment proves non-membership",
			freezeClient: false,
			pathToVerify: bytes.Repeat([]byte{0x01}, 32),
			packets:      []attestations.PacketCompact{{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: make([]byte, 32)}},
			expErr:       nil,
		},
		{
			name:         "failure: non-zero commitment",
			freezeClient: false,
			pathToVerify: bytes.Repeat([]byte{0x01}, 32),
			packets:      []attestations.PacketCompact{{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: bytes.Repeat([]byte{0x01}, 32)}},
			expErr:       attestations.ErrNonMembershipFailed,
		},
		{
			name:         "failure: path not found in attestation",
			freezeClient: false,
			pathToVerify: bytes.Repeat([]byte{0x02}, 32),
			packets:      []attestations.PacketCompact{{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: make([]byte, 32)}},
			expErr:       attestations.ErrNotMember,
		},
		{
			name:         "failure: frozen client",
			freezeClient: true,
			pathToVerify: bytes.Repeat([]byte{0x01}, 32),
			packets:      []attestations.PacketCompact{{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: make([]byte, 32)}},
			expErr:       attestations.ErrClientFrozen,
		},
		{
			name:         "success: multiple packets with same path, all have zero commitments",
			freezeClient: false,
			pathToVerify: bytes.Repeat([]byte{0x01}, 32),
			packets: []attestations.PacketCompact{
				{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: make([]byte, 32)},
				{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: make([]byte, 32)},
			},
			expErr: nil,
		},
		{
			name:         "failure: multiple packets with same path, all have non-zero commitments",
			freezeClient: false,
			pathToVerify: bytes.Repeat([]byte{0x01}, 32),
			packets: []attestations.PacketCompact{
				{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: bytes.Repeat([]byte{0x01}, 32)},
				{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: bytes.Repeat([]byte{0x02}, 32)},
			},
			expErr: attestations.ErrNonMembershipFailed,
		},
		{
			name:         "failure: multiple packets with same path, zero commitment first but non-zero later",
			freezeClient: false,
			pathToVerify: bytes.Repeat([]byte{0x01}, 32),
			packets: []attestations.PacketCompact{
				{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: make([]byte, 32)},
				{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: bytes.Repeat([]byte{0x01}, 32)},
			},
			expErr: attestations.ErrNonMembershipFailed,
		},
		{
			name:         "failure: multiple packets with same path, non-zero commitment first then zero",
			freezeClient: false,
			pathToVerify: bytes.Repeat([]byte{0x01}, 32),
			packets: []attestations.PacketCompact{
				{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: bytes.Repeat([]byte{0x01}, 32)},
				{Path: bytes.Repeat([]byte{0x01}, 32), Commitment: make([]byte, 32)},
			},
			expErr: attestations.ErrNonMembershipFailed,
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
			stateAttestation := s.createStateAttestation(newHeight, newTimestamp)
			signers := []int{0, 1, 2}
			updateProof := s.createAttestationProof(stateAttestation, signers, attestations.AttestationTypeState)

			err := s.lightClientModule.VerifyClientMessage(ctx, clientID, updateProof)
			s.Require().NoError(err)
			_ = s.lightClientModule.UpdateState(ctx, clientID, updateProof)

			if tc.freezeClient {
				s.freezeClient(ctx, clientID)
			}

			hashedPackets := make([]attestations.PacketCompact, len(tc.packets))
			for i, p := range tc.packets {
				hashedPackets[i] = attestations.PacketCompact{
					Path:       crypto.Keccak256(p.Path),
					Commitment: p.Commitment,
				}
			}
			packetAttestation := s.createPacketAttestation(newHeight, hashedPackets)
			nonMembershipProof := s.createAttestationProof(packetAttestation, signers, attestations.AttestationTypePacket)
			nonMembershipProofBz := s.marshalProof(nonMembershipProof)

			proofHeight := clienttypes.NewHeight(0, newHeight)
			err = s.lightClientModule.VerifyNonMembership(ctx, clientID, proofHeight, 0, 0, nonMembershipProofBz, commitmenttypesv2.NewMerklePath(tc.pathToVerify))

			if tc.expErr != nil {
				s.Require().ErrorIs(err, tc.expErr)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
