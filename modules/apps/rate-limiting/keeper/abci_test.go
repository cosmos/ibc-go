// SPDX-License-Identifier: Apache-2.0

package keeper_test

import (
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

// Store a rate limit with a non-zero flow for each duration
func (s *KeeperTestSuite) resetRateLimits(denom string, durations []uint64, nonZeroFlow int64) {
	// Add/reset rate limit with a quota duration hours for each duration in the list
	for i, duration := range durations {
		channelID := fmt.Sprintf("channel-%d", i)

		s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
			Path: &types.Path{
				Denom:             denom,
				ChannelOrClientId: channelID,
			},
			Quota: &types.Quota{
				DurationHours: duration,
			},
			Flow: &types.Flow{
				Inflow:       sdkmath.NewInt(nonZeroFlow),
				Outflow:      sdkmath.NewInt(nonZeroFlow),
				ChannelValue: sdkmath.NewInt(100),
			},
		})
	}
}

func (s *KeeperTestSuite) TestBeginBlocker_NoPanic() {
	err := s.chainA.GetSimApp().RateLimitKeeper.SetHourEpoch(s.chainA.GetContext(), types.HourEpoch{
		Duration: 0,
	})
	s.Require().NoError(err)
	s.Require().NotPanics(func() {
		s.chainA.GetSimApp().RateLimitKeeper.BeginBlocker(s.chainA.GetContext())
	})
}

func (s *KeeperTestSuite) TestBeginBlocker_ReturnsWhenEpochInPast() {
	err := s.chainA.GetSimApp().RateLimitKeeper.SetHourEpoch(s.chainA.GetContext(), types.HourEpoch{
		Duration:       time.Minute,
		EpochStartTime: time.Now().Add(time.Hour * -1),
	})
	s.Require().NoError(err)
	s.Require().NotPanics(func() {
		s.chainA.GetSimApp().RateLimitKeeper.BeginBlocker(s.chainA.GetContext())
	})
}

func (s *KeeperTestSuite) TestBeginBlocker() {
	// We'll create three rate limits with different durations
	// And then pass in epoch ids that will cause each to trigger a reset in order
	// i.e. epochId 2   will only cause duration 2 to trigger (2 % 2 == 0; and 9 % 2 != 0; 25 % 2 != 0),
	//      epochId 9,  will only cause duration 3 to trigger (9 % 2 != 0; and 9 % 3 == 0; 25 % 3 != 0)
	//      epochId 25, will only cause duration 5 to trigger (9 % 5 != 0; and 9 % 5 != 0; 25 % 5 == 0)
	durations := []uint64{2, 3, 5}
	epochIDs := []uint64{2, 9, 25}
	nonZeroFlow := int64(10)

	blockTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	s.coordinator.SetTime(blockTime)

	for i, epochID := range epochIDs {
		// First reset the  rate limits to they have a non-zero flow
		s.resetRateLimits(denom, durations, nonZeroFlow)

		duration := durations[i]
		channelIDFromResetRateLimit := fmt.Sprintf("channel-%d", i)

		// Setup epochs so that the hook triggers
		// (epoch start time + duration must be before block time)
		err := s.chainA.GetSimApp().RateLimitKeeper.SetHourEpoch(s.chainA.GetContext(), types.HourEpoch{
			EpochNumber:    epochID - 1,
			Duration:       time.Minute,
			EpochStartTime: blockTime.Add(-2 * time.Minute),
		})
		s.Require().NoError(err)
		s.chainA.GetSimApp().RateLimitKeeper.BeginBlocker(s.chainA.GetContext())

		// Check rate limits (only one rate limit should reset for each hook trigger)
		rateLimits := s.chainA.GetSimApp().RateLimitKeeper.GetAllRateLimits(s.chainA.GetContext())
		for _, rateLimit := range rateLimits {
			context := fmt.Sprintf("duration: %d, epoch: %d", duration, epochID)

			if rateLimit.Path.ChannelOrClientId == channelIDFromResetRateLimit {
				s.Require().Equal(int64(0), rateLimit.Flow.Inflow.Int64(), "inflow was not reset to 0 - %s", context)
				s.Require().Equal(int64(0), rateLimit.Flow.Outflow.Int64(), "outflow was not reset to 0 - %s", context)
			} else {
				s.Require().Equal(nonZeroFlow, rateLimit.Flow.Inflow.Int64(), "inflow should have been left unchanged - %s", context)
				s.Require().Equal(nonZeroFlow, rateLimit.Flow.Outflow.Int64(), "outflow should have been left unchanged - %s", context)
			}
		}
	}
}

// After a halt the first block moves the epoch past all the missed hours. Each quota is
// reset at most once, and the following blocks must not reset them again.
func (s *KeeperTestSuite) TestBeginBlocker_AfterHalt() {
	// channel-0: 1 hour, channel-1: 4 hours, channel-2: 24 hours
	durations := []uint64{1, 4, 24}
	nonZeroFlow := int64(10)
	keeper := s.chainA.GetSimApp().RateLimitKeeper

	epochStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	err := keeper.SetHourEpoch(s.chainA.GetContext(), types.HourEpoch{
		EpochNumber:    100,
		Duration:       time.Hour,
		EpochStartTime: epochStart,
	})
	s.Require().NoError(err)
	s.resetRateLimits(denom, durations, nonZeroFlow)

	flows := func() map[string]int64 {
		out := map[string]int64{}
		for _, rateLimit := range keeper.GetAllRateLimits(s.chainA.GetContext()) {
			s.Require().Equal(rateLimit.Flow.Inflow, rateLimit.Flow.Outflow)
			out[rateLimit.Path.ChannelOrClientId] = rateLimit.Flow.Outflow.Int64()
		}
		return out
	}

	// epoch 100 ended at 01:00 and the chain was halted until 06:01: epoch 100 -> 106
	blockTime := epochStart.Add(6*time.Hour + time.Minute)
	keeper.BeginBlocker(s.chainA.GetContext().WithBlockTime(blockTime))
	epoch, err := keeper.GetHourEpoch(s.chainA.GetContext())
	s.Require().NoError(err)
	s.Require().Equal(uint64(106), epoch.EpochNumber)
	s.Require().Equal(epochStart.Add(6*time.Hour), epoch.EpochStartTime)
	// the hourly quota and the 4 hour one (period ended at epoch 104) reset once, the daily one did not
	s.Require().Equal(map[string]int64{"channel-0": 0, "channel-1": 0, "channel-2": nonZeroFlow}, flows())

	// new flow in the next blocks is kept until the next hour
	s.resetRateLimits(denom, durations, nonZeroFlow)
	for i := 1; i <= 5; i++ {
		keeper.BeginBlocker(s.chainA.GetContext().WithBlockTime(blockTime.Add(time.Duration(i) * 6 * time.Second)))
		s.Require().Equal(map[string]int64{"channel-0": nonZeroFlow, "channel-1": nonZeroFlow, "channel-2": nonZeroFlow}, flows(), "block %d after the halt", i)
	}
}
