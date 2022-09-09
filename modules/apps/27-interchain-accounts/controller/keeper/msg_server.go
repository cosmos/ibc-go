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
func (k msgServer) RegisterInterchainAccount(goCtx context.Context, msg *types.MsgRegisterInterchainAccount) (*types.MsgRegisterInterchainAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	portID, err := icatypes.NewControllerPortID(msg.Owner)
	if err != nil {
		return nil, err
	}

	channelID, err := k.registerInterchainAccount(ctx, msg.ConnectionId, portID, msg.Version)
	if err != nil {
		return nil, err
	}

	return &types.MsgRegisterInterchainAccountResponse{
		ChannelId: channelID,
	}, nil
}

// SubmitTx defines a rpc handler for MsgSubmitTx
func (k msgServer) SubmitTx(goCtx context.Context, msg *types.MsgSubmitTx) (*types.MsgSubmitTxResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	portID, err := icatypes.NewControllerPortID(msg.Owner)
	if err != nil {
		return nil, err
	}

	// explicitly passing nil as the argument is discarded as the channel capability is retrieved in SendTx.
	seq, err := k.SendTx(ctx, nil, msg.ConnectionId, portID, msg.PacketData, msg.TimeoutTimestamp)
	if err != nil {
		return nil, err
	}

	return &types.MsgSubmitTxResponse{Sequence: seq}, nil
}
