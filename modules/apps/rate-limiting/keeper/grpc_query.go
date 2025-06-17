package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

var _ types.QueryServer = Keeper{}

// Query all rate limits
func (k Keeper) AllRateLimits(c context.Context, req *types.QueryAllRateLimitsRequest) (*types.QueryAllRateLimitsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	rateLimits := k.GetAllRateLimits(ctx)
	return &types.QueryAllRateLimitsResponse{RateLimits: rateLimits}, nil
}

// Query a rate limit by denom and channelId
func (k Keeper) RateLimit(c context.Context, req *types.QueryRateLimitRequest) (*types.QueryRateLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	rateLimit, found := k.GetRateLimit(ctx, req.Denom, req.ChannelOrClientId)
	if !found {
		return &types.QueryRateLimitResponse{}, nil
	}
	return &types.QueryRateLimitResponse{RateLimit: &rateLimit}, nil
}

// Query all rate limits for a given chain
func (k Keeper) RateLimitsByChainID(c context.Context, req *types.QueryRateLimitsByChainIDRequest) (*types.QueryRateLimitsByChainIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	rateLimits := []types.RateLimit{}
	for _, rateLimit := range k.GetAllRateLimits(ctx) {

		// Determine the client state from the channel Id
		_, clientState, err := k.channelKeeper.GetChannelClientState(ctx, transfertypes.PortID, rateLimit.Path.ChannelOrClientId)
		if err != nil {
			var ok bool
			clientState, ok = k.clientKeeper.GetClientState(ctx, rateLimit.Path.ChannelOrClientId)
			if !ok {
				return &types.QueryRateLimitsByChainIDResponse{}, errorsmod.Wrapf(types.ErrInvalidClientState, "Unable to fetch client state from channel or client Id %s", rateLimit.Path.ChannelOrClientId)
			}
		}

		// Check if the client state is a tendermint client
		if clientState.ClientType() == exported.Tendermint {
			// Type assert to tendermint client state
			tmClientState, ok := clientState.(*tmclient.ClientState)
			if !ok {
				// This should never happen if ClientType() == Tendermint, but check anyway
				continue
			}

			// If the chain ID matches, add the rate limit to the returned list
			if tmClientState.GetChainID() == req.ChainId {
				rateLimits = append(rateLimits, rateLimit)
			}
		}
	}

	return &types.QueryRateLimitsByChainIDResponse{RateLimits: rateLimits}, nil
}

// Query all rate limits for a given channel
func (k Keeper) RateLimitsByChannelOrClientID(c context.Context, req *types.QueryRateLimitsByChannelOrClientIDRequest) (*types.QueryRateLimitsByChannelOrClientIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	rateLimits := []types.RateLimit{}
	for _, rateLimit := range k.GetAllRateLimits(ctx) {
		// If the channel ID matches, add the rate limit to the returned list
		if rateLimit.Path.ChannelOrClientId == req.ChannelOrClientId {
			rateLimits = append(rateLimits, rateLimit)
		}
	}

	return &types.QueryRateLimitsByChannelOrClientIDResponse{RateLimits: rateLimits}, nil
}

// Query all blacklisted denoms
func (k Keeper) AllBlacklistedDenoms(c context.Context, req *types.QueryAllBlacklistedDenomsRequest) (*types.QueryAllBlacklistedDenomsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	blacklistedDenoms := k.GetAllBlacklistedDenoms(ctx)
	return &types.QueryAllBlacklistedDenomsResponse{Denoms: blacklistedDenoms}, nil
}

// Query all whitelisted addresses
func (k Keeper) AllWhitelistedAddresses(c context.Context, req *types.QueryAllWhitelistedAddressesRequest) (*types.QueryAllWhitelistedAddressesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	whitelistedAddresses := k.GetAllWhitelistedAddressPairs(ctx)
	return &types.QueryAllWhitelistedAddressesResponse{AddressPairs: whitelistedAddresses}, nil
}
