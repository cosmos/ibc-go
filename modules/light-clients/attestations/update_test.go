package attestations_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v11/modules/core/exported"
	"github.com/cosmos/ibc-go/v11/modules/light-clients/attestations"
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
			proof := s.createAttestationProof(attestationData, tc.signers, attestations.AttestationTypeState)

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
	proof := s.createAttestationProof(attestationData, signers, attestations.AttestationTypeState)

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
	proof := s.createAttestationProof(attestationData, signers, attestations.AttestationTypeState)

	err := s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)
	_ = s.lightClientModule.UpdateState(ctx, clientID, proof)

	s.freezeClient(ctx, clientID)

	status := s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Frozen, status)

	newProofData := s.createStateAttestation(uint64(300), uint64(4*time.Second.Nanoseconds()))
	newProof := s.createAttestationProof(newProofData, signers, attestations.AttestationTypeState)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, newProof)
	s.Require().ErrorIs(err, attestations.ErrClientFrozen)
}

func (s *AttestationsTestSuite) TestUpdateStateOnMisbehaviour() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())
	clientID := testClientID

	testCases := []struct {
		name       string
		initialize bool
		setup      func(ctx sdk.Context, clientID string) exported.ClientMessage
		expPanic   string
		assert     func(ctx sdk.Context, clientID string)
	}{
		{
			name:       "freezes active client",
			initialize: true,
			setup: func(ctx sdk.Context, clientID string) exported.ClientMessage {
				height := uint64(200)
				timestamp := uint64(2 * time.Second.Nanoseconds())
				attestationData := s.createStateAttestation(height, timestamp)
				signers := []int{0, 1, 2}
				proof := s.createAttestationProof(attestationData, signers, attestations.AttestationTypeState)

				err := s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
				s.Require().NoError(err)
				_ = s.lightClientModule.UpdateState(ctx, clientID, proof)

				conflictingData := s.createStateAttestation(height, uint64(3*time.Second.Nanoseconds()))
				return s.createAttestationProof(conflictingData, signers, attestations.AttestationTypeState)
			},
			assert: func(ctx sdk.Context, clientID string) {
				status := s.lightClientModule.Status(ctx, clientID)
				s.Require().Equal(exported.Frozen, status)
			},
		},
		{
			name:       "panic when client not found",
			initialize: false,
			setup: func(_ sdk.Context, _ string) exported.ClientMessage {
				return nil
			},
			expPanic: clienttypes.ErrClientNotFound.Wrap(clientID).Error(),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			ctx := s.chainA.GetContext()
			if tc.initialize {
				s.initializeClient(ctx, clientID, initialHeight, initialTimestamp)
			}

			clientMsg := tc.setup(ctx, clientID)

			if tc.expPanic != "" {
				var panicVal any
				panicFunc := func() {
					defer func() {
						if r := recover(); r != nil {
							panicVal = r
							panic(r)
						}
					}()
					s.lightClientModule.UpdateStateOnMisbehaviour(ctx, clientID, clientMsg)
				}

				s.Require().Panics(panicFunc)
				err, ok := panicVal.(error)
				s.Require().True(ok)
				s.Require().ErrorContains(err, tc.expPanic)
				return
			}

			s.lightClientModule.UpdateStateOnMisbehaviour(ctx, clientID, clientMsg)
			if tc.assert != nil {
				tc.assert(ctx, clientID)
			}
		})
	}
}
