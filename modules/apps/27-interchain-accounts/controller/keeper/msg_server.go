package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the ICS27 MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// RegisterInterchainAccount defines a rpc handler for MsgRegisterInterchainAccount
func (s msgServer) RegisterInterchainAccount(goCtx context.Context, msg *types.MsgRegisterInterchainAccount) (*types.MsgRegisterInterchainAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	portID, err := icatypes.NewControllerPortID(msg.Owner)
	if err != nil {
		return nil, err
	}

	channelID, err := s.registerInterchainAccount(ctx, msg.ConnectionId, portID, msg.Version)
	if err != nil {
		return nil, err
	}

	return &types.MsgRegisterInterchainAccountResponse{
		ChannelId: channelID,
	}, nil
}

// SendTx defines a rpc handler for MsgSendTx
func (s msgServer) SendTx(goCtx context.Context, msg *types.MsgSendTx) (*types.MsgSendTxResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	portID, err := icatypes.NewControllerPortID(msg.Owner)
	if err != nil {
		return nil, err
	}

	// explicitly passing nil as the argument is discarded as the channel capability is retrieved in SendTx.
	absoluteTimeout := uint64(ctx.BlockTime().UnixNano()) + msg.TimeoutTimestamp
	seq, err := s.Keeper.SendTx(ctx, nil, msg.ConnectionId, portID, msg.PacketData, absoluteTimeout)
	if err != nil {
		return nil, err
	}

	return &types.MsgSendTxResponse{Sequence: seq}, nil
}
