package keeper_test

import (
	"time"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// Tests Get/Set Hour epoch
func (s *KeeperTestSuite) TestHourEpoch() {
	expectedHourEpoch := types.HourEpoch{
		Duration:         time.Hour,
		EpochNumber:      1,
		EpochStartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EpochStartHeight: 10,
	}
	err := s.chainA.GetSimApp().RateLimitKeeper.SetHourEpoch(s.chainA.GetContext(), expectedHourEpoch)
	s.Require().NoError(err)

	actualHourEpoch, err := s.chainA.GetSimApp().RateLimitKeeper.GetHourEpoch(s.chainA.GetContext())
	s.Require().NoError(err)
	s.Require().Equal(expectedHourEpoch, actualHourEpoch, "hour epoch")
}

func (s *KeeperTestSuite) TestCheckHourEpochStarting() {
	epochStartTime := time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC)
	blockHeight := int64(10)
	duration := time.Minute

	initialEpoch := types.HourEpoch{
		EpochNumber:    10,
		EpochStartTime: epochStartTime,
		Duration:       duration,
	}
	nextEpoch := types.HourEpoch{
		EpochNumber:      initialEpoch.EpochNumber + 1, // epoch number increments
		EpochStartTime:   epochStartTime.Add(duration), // start time increments by duration
		EpochStartHeight: blockHeight,                  // height gets current block height
		Duration:         duration,
	}

	testCases := []struct {
		name                  string
		blockTime             time.Time
		expectedEpochStarting bool
		initialEpoch          types.HourEpoch
		err                   error
	}{
		{
			name:                  "in middle of epoch",
			blockTime:             epochStartTime.Add(duration / 2), // halfway through epoch
			expectedEpochStarting: false,
			initialEpoch:          initialEpoch,
			err:                   nil,
		},
		{
			name:                  "right before epoch boundary",
			blockTime:             epochStartTime.Add(duration).Add(-1 * time.Second), // 1 second before epoch
			expectedEpochStarting: false,
			initialEpoch:          initialEpoch,
			err:                   nil,
		},
		{
			name:                  "at epoch boundary",
			blockTime:             epochStartTime.Add(duration), // at epoch boundary
			expectedEpochStarting: false,
			initialEpoch:          initialEpoch,
			err:                   nil,
		},
		{
			name:                  "right after epoch boundary",
			blockTime:             epochStartTime.Add(duration).Add(time.Second), // one second after epoch boundary
			expectedEpochStarting: true,
			initialEpoch:          initialEpoch,
			err:                   nil,
		},
		{
			name:                  "in middle of next epoch",
			blockTime:             epochStartTime.Add(duration).Add(duration / 2), // halfway through next epoch
			expectedEpochStarting: true,
			initialEpoch:          initialEpoch,
			err:                   nil,
		},
		{
			name:                  "next epoch skipped",
			blockTime:             epochStartTime.Add(duration * 10), // way after next epoch (still increments only once)
			expectedEpochStarting: true,
			initialEpoch:          initialEpoch,
			err:                   nil,
		},
		{
			name:                  "error - invalid epoch",
			blockTime:             epochStartTime.Add(duration * 10), // way after next epoch (still increments only once)
			expectedEpochStarting: true,
			initialEpoch: types.HourEpoch{
				Duration: 0,
			},
			err: types.ErrInvalidEpoce,
		},
	}

	// Set the block height to blockHeight
	s.coordinator.CommitNBlocks(s.chainA, uint64(blockHeight)-uint64(s.chainA.App.LastBlockHeight()+1))

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.coordinator.SetTime(tc.blockTime)

			err := s.chainA.GetSimApp().RateLimitKeeper.SetHourEpoch(s.chainA.GetContext(), tc.initialEpoch)
			s.Require().NoError(err)

			actualStarting, actualEpochNumber, err := s.chainA.GetSimApp().RateLimitKeeper.CheckHourEpochStarting(s.chainA.GetContext())
			if tc.err != nil {
				s.Require().ErrorIs(err, tc.err)
				return
			}
			s.Require().NoError(err)
			s.Require().Equal(tc.expectedEpochStarting, actualStarting, "epoch starting")

			expectedEpoch := tc.initialEpoch
			if tc.expectedEpochStarting {
				expectedEpoch = nextEpoch
				s.Require().Equal(expectedEpoch.EpochNumber, actualEpochNumber, "epoch number")
			}

			actualHourEpoch, err := s.chainA.GetSimApp().RateLimitKeeper.GetHourEpoch(s.chainA.GetContext())
			s.Require().NoError(err)
			s.Require().Equal(expectedEpoch, actualHourEpoch, "hour epoch")
		})
	}
}
