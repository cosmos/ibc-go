package keeper_test

import (
	"strconv"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const (
	denom     = "denom"
	channelID = "channel-0"
	sender    = "sender"
	receiver  = "receiver"
)

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
	channelID := expectedRateLimit.Path.ChannelOrClientId

	actualRateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denom, channelID)
	s.Require().True(found, "element should have been found, but was not")
	s.Require().Equal(expectedRateLimit, actualRateLimit)
}

func (s *KeeperTestSuite) TestRemoveRateLimit() {
	rateLimits := s.createRateLimits()

	rateLimitToRemove := rateLimits[0]
	denomToRemove := rateLimitToRemove.Path.Denom
	channelIDToRemove := rateLimitToRemove.Path.ChannelOrClientId

	s.chainA.GetSimApp().RateLimitKeeper.RemoveRateLimit(s.chainA.GetContext(), denomToRemove, channelIDToRemove)
	_, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denomToRemove, channelIDToRemove)
	s.Require().False(found, "the removed element should not have been found, but it was")
}

func (s *KeeperTestSuite) TestResetRateLimit() {
	rateLimits := s.createRateLimits()

	rateLimitToReset := rateLimits[0]
	denomToRemove := rateLimitToReset.Path.Denom
	channelIDToRemove := rateLimitToReset.Path.ChannelOrClientId

	err := s.chainA.GetSimApp().RateLimitKeeper.ResetRateLimit(s.chainA.GetContext(), denomToRemove, channelIDToRemove)
	s.Require().NoError(err)

	rateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denomToRemove, channelIDToRemove)
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

func (s *KeeperTestSuite) TestAddRateLimit_ClientId() {
	// Setup client between chain A and chain B
	path := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupClients(path)
	clientID := path.EndpointA.ClientID

	// Mock GetChannelValue to return non-zero
	// Note: This might require adjusting the test suite setup if GetChannelValue isn't easily mockable.
	// For now, assume it works or the underlying bank keeper has supply.
	// A more robust test might involve actually sending tokens.
	// Mint some tokens for the denom to ensure channel value is non-zero
	mintAmount := sdkmath.NewInt(1000)
	mintCoins := sdk.NewCoins(sdk.NewCoin("clientdenom", mintAmount))
	// Revert: Mint back to the transfer module account
	err := s.chainA.GetSimApp().BankKeeper.MintCoins(s.chainA.GetContext(), transfertypes.ModuleName, mintCoins)
	s.Require().NoError(err, "minting coins failed")

	msg := &types.MsgAddRateLimit{
		Signer:            s.chainA.GetSimApp().RateLimitKeeper.GetAuthority(),
		Denom:             "clientdenom",
		ChannelOrClientId: clientID,
		MaxPercentSend:    sdkmath.NewInt(10),
		MaxPercentRecv:    sdkmath.NewInt(10),
		DurationHours:     24,
	}

	// Add the rate limit using the client ID
	err = s.chainA.GetSimApp().RateLimitKeeper.AddRateLimit(s.chainA.GetContext(), msg)
	s.Require().NoError(err, "adding rate limit with client ID should succeed")

	// Verify the rate limit was stored correctly
	_, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), msg.Denom, clientID)
	s.Require().True(found, "rate limit added with client ID should be found")

	// Test adding with an invalid ID (neither channel nor client)
	invalidID := "invalid-id"
	msgInvalid := &types.MsgAddRateLimit{
		Signer:            s.chainA.GetSimApp().RateLimitKeeper.GetAuthority(),
		Denom:             "clientdenom",
		ChannelOrClientId: invalidID,
		MaxPercentSend:    sdkmath.NewInt(10),
		MaxPercentRecv:    sdkmath.NewInt(10),
		DurationHours:     24,
	}
	err = s.chainA.GetSimApp().RateLimitKeeper.AddRateLimit(s.chainA.GetContext(), msgInvalid)
	s.Require().ErrorIs(err, types.ErrChannelNotFound, "adding rate limit with invalid ID should fail")
}
