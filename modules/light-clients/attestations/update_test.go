package attestations_test

import (
	"time"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/cosmos/ibc-go/v10/modules/light-clients/attestations"
)

func (s *AttestationsTestSuite) TestUpdateState() {
	testCases := []struct {
		name    string
		signers []int
		expErr  string
	}{
		{
			name:    "success: sufficient signatures",
			signers: []int{0, 1, 2},
			expErr:  "",
		},
		{
			name:    "failure: insufficient signatures",
			signers: []int{0, 1},
			expErr:  "quorum",
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
			proof := s.createAttestationProof(attestationData, tc.signers)

			err := s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
			if tc.expErr != "" {
				s.Require().ErrorContains(err, tc.expErr)
				return
			}

			s.Require().NoError(err)

			heights := s.lightClientModule.UpdateState(ctx, clientID, proof)
			s.Require().Len(heights, 1)
			s.Require().Equal(newHeight, heights[0].GetRevisionHeight())

			latestHeight := s.lightClientModule.LatestHeight(ctx, clientID)
			s.Require().Equal(newHeight, latestHeight.GetRevisionHeight())

			timestamp, err := s.lightClientModule.TimestampAtHeight(ctx, clientID, latestHeight)
			s.Require().NoError(err)
			s.Require().Equal(newTimestamp, timestamp)
		})
	}
}

func (s *AttestationsTestSuite) TestUpdateStateIdempotency() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())
	clientID := testClientID
	ctx := s.chainA.GetContext()

	s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	attestationData := s.createStateAttestation(newHeight, newTimestamp)
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(attestationData, signers)

	err := s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)

	heights := s.lightClientModule.UpdateState(ctx, clientID, proof)
	s.Require().Len(heights, 1)
	s.Require().Equal(newHeight, heights[0].GetRevisionHeight())

	heights = s.lightClientModule.UpdateState(ctx, clientID, proof)
	s.Require().Len(heights, 1)
	s.Require().Equal(newHeight, heights[0].GetRevisionHeight())

	status := s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Active, status)
}

func (s *AttestationsTestSuite) TestVerifyClientMessageFrozenClient() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())
	clientID := testClientID
	ctx := s.chainA.GetContext()

	s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	attestationData := s.createStateAttestation(newHeight, newTimestamp)
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(attestationData, signers)

	err := s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)
	_ = s.lightClientModule.UpdateState(ctx, clientID, proof)

	s.freezeClient(ctx, clientID)

	status := s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Frozen, status)

	newProofData := s.createStateAttestation(uint64(300), uint64(4*time.Second.Nanoseconds()))
	newProof := s.createAttestationProof(newProofData, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, newProof)
	s.Require().ErrorIs(err, attestations.ErrClientFrozen)
}

func (s *AttestationsTestSuite) TestUpdateStateOnMisbehaviourPanics() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())
	clientID := testClientID
	ctx := s.chainA.GetContext()

	s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)

	status := s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Active, status)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	attestationData := s.createStateAttestation(newHeight, newTimestamp)
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(attestationData, signers)

	err := s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)
	_ = s.lightClientModule.UpdateState(ctx, clientID, proof)

	conflictingTimestamp := uint64(3 * time.Second.Nanoseconds())
	conflictingAttestationData := s.createStateAttestation(newHeight, conflictingTimestamp)
	conflictingProof := s.createAttestationProof(conflictingAttestationData, signers)

	updateStateOnMisbehaviourFunc := func() {
		s.lightClientModule.UpdateStateOnMisbehaviour(ctx, clientID, conflictingProof)
	}
	s.Require().PanicsWithError("updateStateOnMisbehaviour is not supported: invalid request", updateStateOnMisbehaviourFunc)

	status = s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Active, status)
}
