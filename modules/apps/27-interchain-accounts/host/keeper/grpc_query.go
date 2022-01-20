package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/host/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

var _ types.QueryServer = Keeper{}

// Params implements the Query/Params gRPC method
func (q Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := q.GetParams(ctx)

	return &types.QueryParamsResponse{
		Params: &params,
	}, nil
}

// PacketEvents implements the Query/PacketEvents method
func (q Keeper) PacketEvents(goCtx context.Context, req *types.QueryPacketEventsRequest) (*types.QueryPacketEventsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// q.channelKeeper.Get
	msgRecvPacket := channeltypes.NewMsgRecvPacket(channeltypes.Packet{}, nil, clienttypes.Height{}, "")

	return &types.QueryPacketEventsResponse{}, nil
}
