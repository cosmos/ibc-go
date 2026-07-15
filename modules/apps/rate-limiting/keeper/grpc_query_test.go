package keeper_test

import (
	"fmt"
	"time"

	querytypes "github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/keeper"
	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	ibctmtypes "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"
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

// addChainRateLimit registers a tendermint client/connection/channel resolving to
// chainID and stores a rate limit on the given channel and denom.
func (s *KeeperTestSuite) addChainRateLimit(chainID, channelID, denom string) {
	s.T().Helper()

	clientID := "07-tendermint-" + channelID
	connectionID := "connection-" + channelID
	s.chainA.GetSimApp().IBCKeeper.ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, ibctmtypes.NewClientState(
		chainID, ibctmtypes.Fraction{}, 0, 0, 0, clienttypes.Height{}, nil, nil,
	))
	s.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.SetConnection(s.chainA.GetContext(), connectionID, connectiontypes.ConnectionEnd{ClientId: clientID})
	s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), transfertypes.PortID, channelID, channeltypes.Channel{ConnectionHops: []string{connectionID}})
	s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
		Path: &types.Path{Denom: denom, ChannelOrClientId: channelID},
	})
}

// TestPaginatedQueries exercises the pagination contract of the list queries:
// resumable NextKey, and Total that is populated only on a full prefix scan.
func (s *KeeperTestSuite) TestPaginatedQueries() {
	s.Run("all_rate_limits", func() {
		s.SetupTest()
		querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
		expected := s.setupQueryRateLimitTests()

		first, err := querier.AllRateLimits(s.chainA.GetContext(), &types.QueryAllRateLimitsRequest{
			Pagination: &querytypes.PageRequest{Limit: 1},
		})
		s.Require().NoError(err)
		s.Require().Len(first.RateLimits, 1)
		s.Require().NotEmpty(first.Pagination.NextKey)

		rest, err := querier.AllRateLimits(s.chainA.GetContext(), &types.QueryAllRateLimitsRequest{
			Pagination: &querytypes.PageRequest{Key: first.Pagination.NextKey, Limit: 100},
		})
		s.Require().NoError(err)
		s.Require().ElementsMatch(expected, append(first.RateLimits, rest.RateLimits...))
	})

	s.Run("all_blacklisted_denoms", func() {
		s.SetupTest()
		querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
		expected := []string{"denom-A", "denom-B"}
		for _, d := range expected {
			s.chainA.GetSimApp().RateLimitKeeper.AddDenomToBlacklist(s.chainA.GetContext(), d)
		}

		first, err := querier.AllBlacklistedDenoms(s.chainA.GetContext(), &types.QueryAllBlacklistedDenomsRequest{
			Pagination: &querytypes.PageRequest{Limit: 1},
		})
		s.Require().NoError(err)
		s.Require().Len(first.Denoms, 1)
		s.Require().NotEmpty(first.Pagination.NextKey)

		rest, err := querier.AllBlacklistedDenoms(s.chainA.GetContext(), &types.QueryAllBlacklistedDenomsRequest{
			Pagination: &querytypes.PageRequest{Key: first.Pagination.NextKey, Limit: 100},
		})
		s.Require().NoError(err)
		s.Require().ElementsMatch(expected, append(first.Denoms, rest.Denoms...))
	})

	s.Run("all_whitelisted_addresses", func() {
		s.SetupTest()
		querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
		expected := []types.WhitelistedAddressPair{
			{Sender: "address-A", Receiver: "address-B"},
			{Sender: "address-C", Receiver: "address-D"},
		}
		for _, pair := range expected {
			s.chainA.GetSimApp().RateLimitKeeper.SetWhitelistedAddressPair(s.chainA.GetContext(), pair)
		}

		first, err := querier.AllWhitelistedAddresses(s.chainA.GetContext(), &types.QueryAllWhitelistedAddressesRequest{
			Pagination: &querytypes.PageRequest{Limit: 1},
		})
		s.Require().NoError(err)
		s.Require().Len(first.AddressPairs, 1)
		s.Require().NotEmpty(first.Pagination.NextKey)

		rest, err := querier.AllWhitelistedAddresses(s.chainA.GetContext(), &types.QueryAllWhitelistedAddressesRequest{
			Pagination: &querytypes.PageRequest{Key: first.Pagination.NextKey, Limit: 100},
		})
		s.Require().NoError(err)
		s.Require().ElementsMatch(expected, append(first.AddressPairs, rest.AddressPairs...))
	})

	s.Run("rate_limits_by_chain_id", func() {
		s.SetupTest()
		querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
		const target = "chain-target"
		s.addChainRateLimit(target, "channel-1", "denom-a")
		s.addChainRateLimit(target, "channel-2", "denom-b")
		s.addChainRateLimit("chain-other", "channel-3", "denom-c")

		first, err := querier.RateLimitsByChainID(s.chainA.GetContext(), &types.QueryRateLimitsByChainIDRequest{
			ChainId:    target,
			Pagination: &querytypes.PageRequest{Limit: 1},
		})
		s.Require().NoError(err)
		s.Require().Len(first.RateLimits, 1)
		s.Require().NotEmpty(first.Pagination.NextKey)

		rest, err := querier.RateLimitsByChainID(s.chainA.GetContext(), &types.QueryRateLimitsByChainIDRequest{
			ChainId:    target,
			Pagination: &querytypes.PageRequest{Key: first.Pagination.NextKey, Limit: 100},
		})
		s.Require().NoError(err)

		got := make([]types.RateLimit, 0, len(first.RateLimits)+len(rest.RateLimits))
		got = append(got, first.RateLimits...)
		got = append(got, rest.RateLimits...)
		s.Require().Len(got, 2)
		denoms := []string{got[0].Path.Denom, got[1].Path.Denom}
		s.Require().ElementsMatch([]string{"denom-a", "denom-b"}, denoms)
	})

	s.Run("rate_limits_by_channel_or_client_id", func() {
		s.SetupTest()
		querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
		const target = "channel-target"
		s.addChainRateLimit("chain-1", target, "denom-a")
		// Same channel, different denom -> distinct rate-limit entry.
		s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
			Path: &types.Path{Denom: "denom-b", ChannelOrClientId: target},
		})
		s.addChainRateLimit("chain-2", "channel-other", "denom-c")

		first, err := querier.RateLimitsByChannelOrClientID(s.chainA.GetContext(), &types.QueryRateLimitsByChannelOrClientIDRequest{
			ChannelOrClientId: target,
			Pagination:        &querytypes.PageRequest{Limit: 1},
		})
		s.Require().NoError(err)
		s.Require().Len(first.RateLimits, 1)
		s.Require().NotEmpty(first.Pagination.NextKey)

		rest, err := querier.RateLimitsByChannelOrClientID(s.chainA.GetContext(), &types.QueryRateLimitsByChannelOrClientIDRequest{
			ChannelOrClientId: target,
			Pagination:        &querytypes.PageRequest{Key: first.Pagination.NextKey, Limit: 100},
		})
		s.Require().NoError(err)

		got := make([]types.RateLimit, 0, len(first.RateLimits)+len(rest.RateLimits))
		got = append(got, first.RateLimits...)
		got = append(got, rest.RateLimits...)
		s.Require().Len(got, 2)
		denoms := []string{got[0].Path.Denom, got[1].Path.Denom}
		s.Require().ElementsMatch([]string{"denom-a", "denom-b"}, denoms)
	})

	s.Run("count_total_omitted_for_key_resumed_pages", func() {
		s.SetupTest()
		querier := keeper.NewQuerier(s.chainA.GetSimApp().RateLimitKeeper)
		const target = "channel-total-target"
		s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{Path: &types.Path{Denom: "denom-a", ChannelOrClientId: target}})
		s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{Path: &types.Path{Denom: "denom-b", ChannelOrClientId: target}})
		s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{Path: &types.Path{Denom: "denom-c", ChannelOrClientId: "channel-other"}})

		first, err := querier.RateLimitsByChannelOrClientID(s.chainA.GetContext(), &types.QueryRateLimitsByChannelOrClientIDRequest{
			ChannelOrClientId: target,
			Pagination:        &querytypes.PageRequest{Limit: 1, CountTotal: true},
		})
		s.Require().NoError(err)
		s.Require().Equal(uint64(2), first.Pagination.Total, "full-scan page should report Total")

		second, err := querier.RateLimitsByChannelOrClientID(s.chainA.GetContext(), &types.QueryRateLimitsByChannelOrClientIDRequest{
			ChannelOrClientId: target,
			Pagination:        &querytypes.PageRequest{Key: first.Pagination.NextKey, Limit: 1, CountTotal: true},
		})
		s.Require().NoError(err)
		s.Require().Equal(uint64(0), second.Pagination.Total, "key-resumed page should omit Total")
	})
}
