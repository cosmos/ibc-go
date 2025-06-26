package keeper_test

import (
	"fmt"
	"time"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctmtypes "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

// Add three rate limits on different channels
// Each should have a different chainId
func (s *KeeperTestSuite) setupQueryRateLimitTests() []types.RateLimit {
	s.T().Helper()

	rateLimits := []types.RateLimit{}
	for i := range int64(3) {
		clientID := fmt.Sprintf("07-tendermint-%d", i)
		chainID := fmt.Sprintf("chain-%d", i)
		connectionID := fmt.Sprintf("connection-%d", i)
		channelID := fmt.Sprintf("channel-%d", i)

		// First register the client, connection, and channel (so we can map back to chainId)
		// Nothing in the client state matters besides the chainId
		clientState := ibctmtypes.NewClientState(chainID, ibctmtypes.Fraction{}, time.Duration(0), time.Duration(0), time.Duration(0), clienttypes.Height{}, nil, nil)
		connection := connectiontypes.ConnectionEnd{ClientId: clientID}
		channel := channeltypes.Channel{ConnectionHops: []string{connectionID}}

		s.chainA.GetSimApp().IBCKeeper.ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)
		s.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.SetConnection(s.chainA.GetContext(), connectionID, connection)
		s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), transfertypes.PortID, channelID, channel)

		// Then add the rate limit
		rateLimit := types.RateLimit{
			Path: &types.Path{Denom: "denom", ChannelOrClientId: channelID},
		}
		s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), rateLimit)
		rateLimits = append(rateLimits, rateLimit)
	}
	return rateLimits
}

func (s *KeeperTestSuite) TestQueryAllRateLimits() {
	querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
	expectedRateLimits := s.setupQueryRateLimitTests()
	queryResponse, err := querier.AllRateLimits(s.chainA.GetContext(), &types.QueryAllRateLimitsRequest{})
	s.Require().NoError(err)
	s.Require().ElementsMatch(expectedRateLimits, queryResponse.RateLimits)
}

func (s *KeeperTestSuite) TestQueryRateLimit() {
	querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
	allRateLimits := s.setupQueryRateLimitTests()
	for _, expectedRateLimit := range allRateLimits {
		queryResponse, err := querier.RateLimit(s.chainA.GetContext(), &types.QueryRateLimitRequest{
			Denom:             expectedRateLimit.Path.Denom,
			ChannelOrClientId: expectedRateLimit.Path.ChannelOrClientId,
		})
		s.Require().NoError(err, "no error expected when querying rate limit on channel: %s", expectedRateLimit.Path.ChannelOrClientId)
		s.Require().Equal(expectedRateLimit, *queryResponse.RateLimit)
	}
}

func (s *KeeperTestSuite) TestQueryRateLimitsByChainId() {
	querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
	allRateLimits := s.setupQueryRateLimitTests()
	for i, expectedRateLimit := range allRateLimits {
		chainID := fmt.Sprintf("chain-%d", i)
		queryResponse, err := querier.RateLimitsByChainID(s.chainA.GetContext(), &types.QueryRateLimitsByChainIDRequest{
			ChainId: chainID,
		})
		s.Require().NoError(err, "no error expected when querying rate limit on chain: %s", chainID)
		s.Require().Len(queryResponse.RateLimits, 1)
		s.Require().Equal(expectedRateLimit, queryResponse.RateLimits[0])
	}
}

func (s *KeeperTestSuite) TestQueryRateLimitsByChannelOrClientId() {
	querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
	allRateLimits := s.setupQueryRateLimitTests()
	for i, expectedRateLimit := range allRateLimits {
		channelID := fmt.Sprintf("channel-%d", i)
		queryResponse, err := querier.RateLimitsByChannelOrClientID(s.chainA.GetContext(), &types.QueryRateLimitsByChannelOrClientIDRequest{
			ChannelOrClientId: channelID,
		})
		s.Require().NoError(err, "no error expected when querying rate limit on channel: %s", channelID)
		s.Require().Len(queryResponse.RateLimits, 1)
		s.Require().Equal(expectedRateLimit, queryResponse.RateLimits[0])
	}
}

func (s *KeeperTestSuite) TestQueryAllBlacklistedDenoms() {
	querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
	s.chainA.GetSimApp().RateLimitKeeper.AddDenomToBlacklist(s.chainA.GetContext(), "denom-A")
	s.chainA.GetSimApp().RateLimitKeeper.AddDenomToBlacklist(s.chainA.GetContext(), "denom-B")

	queryResponse, err := querier.AllBlacklistedDenoms(s.chainA.GetContext(), &types.QueryAllBlacklistedDenomsRequest{})
	s.Require().NoError(err, "no error expected when querying blacklisted denoms")
	s.Require().Equal([]string{"denom-A", "denom-B"}, queryResponse.Denoms)
}

func (s *KeeperTestSuite) TestQueryAllWhitelistedAddresses() {
	querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
	s.chainA.GetSimApp().RateLimitKeeper.SetWhitelistedAddressPair(s.chainA.GetContext(), types.WhitelistedAddressPair{
		Sender:   "address-A",
		Receiver: "address-B",
	})
	s.chainA.GetSimApp().RateLimitKeeper.SetWhitelistedAddressPair(s.chainA.GetContext(), types.WhitelistedAddressPair{
		Sender:   "address-C",
		Receiver: "address-D",
	})
	queryResponse, err := querier.AllWhitelistedAddresses(s.chainA.GetContext(), &types.QueryAllWhitelistedAddressesRequest{})
	s.Require().NoError(err, "no error expected when querying whitelisted addresses")

	expectedWhitelist := []types.WhitelistedAddressPair{
		{Sender: "address-A", Receiver: "address-B"},
		{Sender: "address-C", Receiver: "address-D"},
	}
	s.Require().Equal(expectedWhitelist, queryResponse.AddressPairs)
}
