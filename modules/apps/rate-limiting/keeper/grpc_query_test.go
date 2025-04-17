package keeper_test

import (
	// "context" // Removed unused import
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types" // Add sdk import
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
	rateLimits := []types.RateLimit{}
	for i := int64(0); i <= 2; i++ {
		clientId := fmt.Sprintf("07-tendermint-%d", i)
		chainId := fmt.Sprintf("chain-%d", i)
		connectionId := fmt.Sprintf("connection-%d", i)
		channelId := fmt.Sprintf("channel-%d", i)

		// First register the client, connection, and channel (so we can map back to chainId)
		// Nothing in the client state matters besides the chainId
		clientState := ibctmtypes.NewClientState(
			chainId, ibctmtypes.Fraction{}, time.Duration(0), time.Duration(0), time.Duration(0), clienttypes.Height{}, nil, nil,
		)
		connection := connectiontypes.ConnectionEnd{ClientId: clientId}
		channel := channeltypes.Channel{ConnectionHops: []string{connectionId}}

		s.chainA.GetSimApp().IBCKeeper.ClientKeeper.SetClientState(s.chainA.GetContext(), clientId, clientState)
		s.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.SetConnection(s.chainA.GetContext(), connectionId, connection)
		s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), transfertypes.PortID, channelId, channel)

		// Then add the rate limit
		rateLimit := types.RateLimit{
			Path: &types.Path{Denom: "denom", ChannelOrClientId: channelId},
		}
		s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), rateLimit)
		rateLimits = append(rateLimits, rateLimit)
	}
	return rateLimits
}

func (s *KeeperTestSuite) TestQueryAllRateLimits() {
	expectedRateLimits := s.setupQueryRateLimitTests()
	queryResponse, err := s.chainA.GetSimApp().RateLimitKeeper.AllRateLimits(sdk.WrapSDKContext(s.chainA.GetContext()), &types.QueryAllRateLimitsRequest{}) // Wrap context
	s.Require().NoError(err)
	s.Require().ElementsMatch(expectedRateLimits, queryResponse.RateLimits)
}

func (s *KeeperTestSuite) TestQueryRateLimit() {
	allRateLimits := s.setupQueryRateLimitTests()
	for _, expectedRateLimit := range allRateLimits {
		queryResponse, err := s.chainA.GetSimApp().RateLimitKeeper.RateLimit(sdk.WrapSDKContext(s.chainA.GetContext()), &types.QueryRateLimitRequest{ // Wrap context
			Denom:             expectedRateLimit.Path.Denom,
			ChannelOrClientId: expectedRateLimit.Path.ChannelOrClientId,
		})
		s.Require().NoError(err, "no error expected when querying rate limit on channel: %s", expectedRateLimit.Path.ChannelOrClientId)
		s.Require().Equal(expectedRateLimit, *queryResponse.RateLimit)
	}
}

func (s *KeeperTestSuite) TestQueryRateLimitsByChainId() {
	allRateLimits := s.setupQueryRateLimitTests()
	for i, expectedRateLimit := range allRateLimits {
		chainId := fmt.Sprintf("chain-%d", i)
		queryResponse, err := s.chainA.GetSimApp().RateLimitKeeper.RateLimitsByChainId(sdk.WrapSDKContext(s.chainA.GetContext()), &types.QueryRateLimitsByChainIdRequest{ // Wrap context
			ChainId: chainId,
		})
		s.Require().NoError(err, "no error expected when querying rate limit on chain: %s", chainId)
		s.Require().Len(queryResponse.RateLimits, 1)
		s.Require().Equal(expectedRateLimit, queryResponse.RateLimits[0])
	}
}

func (s *KeeperTestSuite) TestQueryRateLimitsByChannelOrClientId() {
	allRateLimits := s.setupQueryRateLimitTests()
	for i, expectedRateLimit := range allRateLimits {
		channelId := fmt.Sprintf("channel-%d", i)
		queryResponse, err := s.chainA.GetSimApp().RateLimitKeeper.RateLimitsByChannelOrClientId(sdk.WrapSDKContext(s.chainA.GetContext()), &types.QueryRateLimitsByChannelOrClientIdRequest{ // Wrap context
			ChannelOrClientId: channelId,
		})
		s.Require().NoError(err, "no error expected when querying rate limit on channel: %s", channelId)
		s.Require().Len(queryResponse.RateLimits, 1)
		s.Require().Equal(expectedRateLimit, queryResponse.RateLimits[0])
	}
}

func (s *KeeperTestSuite) TestQueryAllBlacklistedDenoms() {
	s.chainA.GetSimApp().RateLimitKeeper.AddDenomToBlacklist(s.chainA.GetContext(), "denom-A")
	s.chainA.GetSimApp().RateLimitKeeper.AddDenomToBlacklist(s.chainA.GetContext(), "denom-B")

	queryResponse, err := s.chainA.GetSimApp().RateLimitKeeper.AllBlacklistedDenoms(sdk.WrapSDKContext(s.chainA.GetContext()), &types.QueryAllBlacklistedDenomsRequest{}) // Wrap context
	s.Require().NoError(err, "no error expected when querying blacklisted denoms")
	s.Require().Equal([]string{"denom-A", "denom-B"}, queryResponse.Denoms)
}

func (s *KeeperTestSuite) TestQueryAllWhitelistedAddresses() {
	s.chainA.GetSimApp().RateLimitKeeper.SetWhitelistedAddressPair(s.chainA.GetContext(), types.WhitelistedAddressPair{
		Sender:   "address-A",
		Receiver: "address-B",
	})
	s.chainA.GetSimApp().RateLimitKeeper.SetWhitelistedAddressPair(s.chainA.GetContext(), types.WhitelistedAddressPair{
		Sender:   "address-C",
		Receiver: "address-D",
	})
	queryResponse, err := s.chainA.GetSimApp().RateLimitKeeper.AllWhitelistedAddresses(sdk.WrapSDKContext(s.chainA.GetContext()), &types.QueryAllWhitelistedAddressesRequest{}) // Wrap context
	s.Require().NoError(err, "no error expected when querying whitelisted addresses")

	expectedWhitelist := []types.WhitelistedAddressPair{
		{Sender: "address-A", Receiver: "address-B"},
		{Sender: "address-C", Receiver: "address-D"},
	}
	s.Require().Equal(expectedWhitelist, queryResponse.AddressPairs)
}
