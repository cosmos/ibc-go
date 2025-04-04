package keeper_test

import (
	"strconv"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"

	sdkmath "cosmossdk.io/math"
)

const (
	denom     = "denom"
	channelId = "channel-0"
	sender    = "sender"
	receiver  = "receiver"
)

type action struct {
	direction           types.PacketDirection
	amount              int64
	addToBlacklist      bool
	removeFromBlacklist bool
	addToWhitelist      bool
	removeFromWhitelist bool
	skipFlowUpdate      bool
	expectedError       string
}

type checkRateLimitTestCase struct {
	name    string
	actions []action
}

// Helper function to create 5 rate limit objects with various attributes
func (s *KeeperTestSuite) createRateLimits() []types.RateLimit {
	rateLimits := []types.RateLimit{}
	for i := 1; i <= 5; i++ {
		suffix := strconv.Itoa(i)
		rateLimit := types.RateLimit{
			Path: &types.Path{Denom: "denom-" + suffix, ChannelOrClientId: "channel-" + suffix},
			Flow: &types.Flow{Inflow: sdkmath.NewInt(10), Outflow: sdkmath.NewInt(10)},
		}

		rateLimits = append(rateLimits, rateLimit)
		s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), rateLimit)
	}
	return rateLimits
}

func (s *KeeperTestSuite) TestGetRateLimit() {
	rateLimits := s.createRateLimits()

	expectedRateLimit := rateLimits[0]
	denom := expectedRateLimit.Path.Denom
	channelId := expectedRateLimit.Path.ChannelOrClientId

	actualRateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denom, channelId)
	s.Require().True(found, "element should have been found, but was not")
	s.Require().Equal(expectedRateLimit, actualRateLimit)
}

func (s *KeeperTestSuite) TestRemoveRateLimit() {
	rateLimits := s.createRateLimits()

	rateLimitToRemove := rateLimits[0]
	denomToRemove := rateLimitToRemove.Path.Denom
	channelIdToRemove := rateLimitToRemove.Path.ChannelOrClientId

	s.chainA.GetSimApp().RateLimitKeeper.RemoveRateLimit(s.chainA.GetContext(), denomToRemove, channelIdToRemove)
	_, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denomToRemove, channelIdToRemove)
	s.Require().False(found, "the removed element should not have been found, but it was")
}

func (s *KeeperTestSuite) TestResetRateLimit() {
	rateLimits := s.createRateLimits()

	rateLimitToReset := rateLimits[0]
	denomToRemove := rateLimitToReset.Path.Denom
	channelIdToRemove := rateLimitToReset.Path.ChannelOrClientId

	err := s.chainA.GetSimApp().RateLimitKeeper.ResetRateLimit(s.chainA.GetContext(), denomToRemove, channelIdToRemove)
	s.Require().NoError(err)

	rateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denomToRemove, channelIdToRemove)
	s.Require().True(found, "element should have been found, but was not")
	s.Require().Zero(rateLimit.Flow.Inflow.Int64(), "Inflow should have been reset to 0")
	s.Require().Zero(rateLimit.Flow.Outflow.Int64(), "Outflow should have been reset to 0")
}

func (s *KeeperTestSuite) TestGetAllRateLimits() {
	expectedRateLimits := s.createRateLimits()
	actualRateLimits := s.chainA.GetSimApp().RateLimitKeeper.GetAllRateLimits(s.chainA.GetContext())
	s.Require().Len(actualRateLimits, len(expectedRateLimits))
	s.Require().ElementsMatch(expectedRateLimits, actualRateLimits, "all rate limits")
}