package attestations_test

import (
	"bytes"
	"crypto/sha256"
	"strings"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	attestations "github.com/cosmos/ibc-go/v10/modules/light-clients/10-attestations"
)

func (s *AttestationsTestSuite) TestClientStateValidate() {
	testCases := []struct {
		name        string
		clientState *attestations.ClientState
		expErr      bool
	}{
		{
			"valid client state",
			s.createClientState(1),
			false,
		},
		{
			"zero latest height",
			attestations.NewClientState(s.attestorAddrs, s.minRequiredSigs, 0),
			true,
		},
		{
			"empty attestor addresses",
			attestations.NewClientState([]string{}, 1, 1),
			true,
		},
		{
			"zero min required sigs",
			attestations.NewClientState(s.attestorAddrs, 0, 1),
			true,
		},
		{
			"min required sigs exceeds attestor count",
			attestations.NewClientState(s.attestorAddrs, 10, 1),
			true,
		},
		{
			"duplicate attestor address",
			attestations.NewClientState([]string{s.attestorAddrs[0], s.attestorAddrs[0]}, 1, 1),
			true,
		},
		{
			"empty attestor address",
			attestations.NewClientState([]string{""}, 1, 1),
			true,
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
			"invalid address format - not hex",
			attestations.NewClientState([]string{"not-a-valid-address"}, 1, 1),
			"invalid attestor address format",
		},
		{
			"invalid address format - too short",
			attestations.NewClientState([]string{"0x1234"}, 1, 1),
			"invalid attestor address format",
		},
		{
			"valid checksummed address",
			attestations.NewClientState([]string{s.attestorAddrs[0]}, 1, 1),
			"",
		},
		{
			"valid lowercase address",
			attestations.NewClientState([]string{strings.ToLower(s.attestorAddrs[0])}, 1, 1),
			"",
		},
		{
			"duplicate addresses with different case",
			attestations.NewClientState([]string{s.attestorAddrs[0], strings.ToLower(s.attestorAddrs[0])}, 1, 1),
			"duplicate attestor address",
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
		commitment   []byte
		expErr       error
	}{
		{
			"success: matching path and commitment",
			false,
			bytes.Repeat([]byte{0x01}, 32),
			bytes.Repeat([]byte{0x01}, 32),
			bytes.Repeat([]byte{0xAB}, 32),
			nil,
		},
		{
			"failure: mismatched path",
			false,
			bytes.Repeat([]byte{0x02}, 32),
			bytes.Repeat([]byte{0x01}, 32),
			bytes.Repeat([]byte{0xCD}, 32),
			attestations.ErrNotMember,
		},
		{
			"failure: frozen client",
			true,
			bytes.Repeat([]byte{0x01}, 32),
			bytes.Repeat([]byte{0x01}, 32),
			bytes.Repeat([]byte{0xAB}, 32),
			attestations.ErrClientFrozen,
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
			updateProof := s.createAttestationProof(stateAttestation, signers)

			err := s.lightClientModule.VerifyClientMessage(ctx, clientID, updateProof)
			s.Require().NoError(err)

			heights := s.lightClientModule.UpdateState(ctx, clientID, updateProof)
			s.Require().Len(heights, 1)

			if tc.freezeClient {
				s.freezeClient(ctx, clientID)
			}

			packetAttestation := s.createPacketAttestation(newHeight, []attestations.PacketCompact{{Path: tc.attestedPath, Commitment: tc.commitment}})
			membershipProof := s.createAttestationProof(packetAttestation, signers)
			membershipProofBz := s.marshalProof(membershipProof)

			proofHeight := clienttypes.NewHeight(0, newHeight)
			err = s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, membershipProofBz, bytePath(tc.pathToVerify), tc.commitment)

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
	path := bytePath([]byte("key"))
	value := []byte("value")

	malformedProof := []byte("invalid-protobuf-data")
	err := s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, malformedProof, path, value)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "failed to unmarshal proof")

	malformedData := []byte("invalid-packet-attestation-data")
	proof := s.createAttestationProof(malformedData, []int{0, 1, 2})
	proofBz := s.marshalProof(proof)

	err = s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, proofBz, path, value)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "failed to unmarshal attestation data")
}

func (s *AttestationsTestSuite) TestVerifyMembershipWithKeyPath() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())
	clientID := testClientID
	ctx := s.chainA.GetContext()

	s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)

	path := mockPath{}
	pathBytes := []byte("key/path")
	hashedPath := sha256.Sum256(pathBytes)

	value32 := make([]byte, 32)
	copy(value32, []byte("value"))

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())

	packetAttestation := s.createPacketAttestation(newHeight, []attestations.PacketCompact{{Path: hashedPath[:], Commitment: value32}})
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(packetAttestation, signers)
	proofBz := s.marshalProof(proof)

	s.updateClientState(ctx, clientID, newHeight, newTimestamp)

	proofHeight := clienttypes.NewHeight(0, newHeight)
	err := s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, proofBz, path, value32)
	s.Require().NoError(err)
}

func (s *AttestationsTestSuite) TestVerifyNonMembership() {
	testCases := []struct {
		name         string
		freezeClient bool
		pathToVerify []byte
		attestedPath []byte
		commitment   []byte
		expErr       error
	}{
		{
			"success: zero commitment proves non-membership",
			false,
			bytes.Repeat([]byte{0x01}, 32),
			bytes.Repeat([]byte{0x01}, 32),
			make([]byte, 32),
			nil,
		},
		{
			"failure: non-zero commitment",
			false,
			bytes.Repeat([]byte{0x01}, 32),
			bytes.Repeat([]byte{0x01}, 32),
			bytes.Repeat([]byte{0x01}, 32),
			attestations.ErrNonMembershipFailed,
		},
		{
			"failure: path not found in attestation",
			false,
			bytes.Repeat([]byte{0x02}, 32),
			bytes.Repeat([]byte{0x01}, 32),
			make([]byte, 32),
			attestations.ErrNotMember,
		},
		{
			"failure: frozen client",
			true,
			bytes.Repeat([]byte{0x01}, 32),
			bytes.Repeat([]byte{0x01}, 32),
			make([]byte, 32),
			attestations.ErrClientFrozen,
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
			updateProof := s.createAttestationProof(stateAttestation, signers)

			err := s.lightClientModule.VerifyClientMessage(ctx, clientID, updateProof)
			s.Require().NoError(err)
			_ = s.lightClientModule.UpdateState(ctx, clientID, updateProof)

			if tc.freezeClient {
				s.freezeClient(ctx, clientID)
			}

			packetAttestation := s.createPacketAttestation(newHeight, []attestations.PacketCompact{{Path: tc.attestedPath, Commitment: tc.commitment}})
			nonMembershipProof := s.createAttestationProof(packetAttestation, signers)
			nonMembershipProofBz := s.marshalProof(nonMembershipProof)

			proofHeight := clienttypes.NewHeight(0, newHeight)
			err = s.lightClientModule.VerifyNonMembership(ctx, clientID, proofHeight, 0, 0, nonMembershipProofBz, bytePath(tc.pathToVerify))

			if tc.expErr != nil {
				s.Require().ErrorIs(err, tc.expErr)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
