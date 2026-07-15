package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	tmclient "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	k *Keeper
}

func NewQuerier(keeper *Keeper) Querier {
	return Querier{k: keeper}
}

// Query all rate limits
func (k Querier) AllRateLimits(c context.Context, req *types.QueryAllRateLimitsRequest) (*types.QueryAllRateLimitsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := k.k.prefixedStore(ctx, types.RateLimitKeyPrefix)

	rateLimits := []types.RateLimit{}
	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		rateLimit := types.RateLimit{}
		k.k.cdc.MustUnmarshal(value, &rateLimit)
		rateLimits = append(rateLimits, rateLimit)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryAllRateLimitsResponse{RateLimits: rateLimits, Pagination: pageRes}, nil
}

// Query a rate limit by denom and channelId
func (k Querier) RateLimit(c context.Context, req *types.QueryRateLimitRequest) (*types.QueryRateLimitResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	rateLimit, found := k.k.GetRateLimit(ctx, req.Denom, req.ChannelOrClientId)
	if !found {
		return &types.QueryRateLimitResponse{}, nil
	}
	return &types.QueryRateLimitResponse{RateLimit: &rateLimit}, nil
}

// RateLimitsByChainID returns paginated rate limits whose channel/client resolves to the given chain.
func (k Querier) RateLimitsByChainID(c context.Context, req *types.QueryRateLimitsByChainIDRequest) (*types.QueryRateLimitsByChainIDResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.ChainId == "" {
		return nil, status.Error(codes.InvalidArgument, "chain_id cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := k.k.prefixedStore(ctx, types.RateLimitKeyPrefix)

	rateLimits := []types.RateLimit{}
	pageRes, err := query.FilteredPaginate(store, req.Pagination, func(_, value []byte, accumulate bool) (bool, error) {
		var rateLimit types.RateLimit
		k.k.cdc.MustUnmarshal(value, &rateLimit)

		// Determine the client state from the channel Id
		_, clientState, err := k.k.channelKeeper.GetChannelClientState(ctx, transfertypes.PortID, rateLimit.Path.ChannelOrClientId)
		if err != nil {
			var ok bool
			clientState, ok = k.k.clientKeeper.GetClientState(ctx, rateLimit.Path.ChannelOrClientId)
			if !ok {
				return false, errorsmod.Wrapf(types.ErrInvalidClientState, "unable to fetch client state from channel or client id")
			}
		}
		client, ok := clientState.(*tmclient.ClientState)
		if !ok {
			// If the client state is not a tendermint client state, skip this rate limit
			return false, nil
		}
		if client.ChainId != req.ChainId {
			return false, nil
		}
		if accumulate {
			rateLimits = append(rateLimits, rateLimit)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryRateLimitsByChainIDResponse{RateLimits: rateLimits, Pagination: pageRes}, nil
}

// RateLimitsByChannelOrClientID returns paginated rate limits for the given channel or client ID.
func (k Querier) RateLimitsByChannelOrClientID(c context.Context, req *types.QueryRateLimitsByChannelOrClientIDRequest) (*types.QueryRateLimitsByChannelOrClientIDResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.ChannelOrClientId == "" {
		return nil, status.Error(codes.InvalidArgument, "channel_or_client_id cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := k.k.prefixedStore(ctx, types.RateLimitKeyPrefix)

	rateLimits := []types.RateLimit{}
	pageRes, err := query.FilteredPaginate(store, req.Pagination, func(_, value []byte, accumulate bool) (bool, error) {
		var rateLimit types.RateLimit
		k.k.cdc.MustUnmarshal(value, &rateLimit)

		if rateLimit.Path.ChannelOrClientId != req.ChannelOrClientId {
			return false, nil
		}
		if accumulate {
			rateLimits = append(rateLimits, rateLimit)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryRateLimitsByChannelOrClientIDResponse{RateLimits: rateLimits, Pagination: pageRes}, nil
}

// Query all blacklisted denoms
func (k Querier) AllBlacklistedDenoms(c context.Context, req *types.QueryAllBlacklistedDenomsRequest) (*types.QueryAllBlacklistedDenomsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := k.k.prefixedStore(ctx, types.DenomBlacklistKeyPrefix)

	blacklistedDenoms := []string{}
	pageRes, err := query.Paginate(store, req.Pagination, func(key, _ []byte) error {
		blacklistedDenoms = append(blacklistedDenoms, string(key))
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryAllBlacklistedDenomsResponse{Denoms: blacklistedDenoms, Pagination: pageRes}, nil
}

// Query all whitelisted addresses
func (k Querier) AllWhitelistedAddresses(c context.Context, req *types.QueryAllWhitelistedAddressesRequest) (*types.QueryAllWhitelistedAddressesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := k.k.prefixedStore(ctx, types.AddressWhitelistKeyPrefix)

	whitelistedAddresses := []types.WhitelistedAddressPair{}
	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		whitelist := types.WhitelistedAddressPair{}
		k.k.cdc.MustUnmarshal(value, &whitelist)
		whitelistedAddresses = append(whitelistedAddresses, whitelist)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryAllWhitelistedAddressesResponse{AddressPairs: whitelistedAddresses, Pagination: pageRes}, nil
}
