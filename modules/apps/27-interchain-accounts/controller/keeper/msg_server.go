package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

var _ types.MsgServer = Keeper{}

<<<<<<< HEAD
// RegisterAccount defines a rpc handler for MsgRegisterAccount
func (k Keeper) RegisterAccount(goCtx context.Context, msg *types.MsgRegisterAccount) (*types.MsgRegisterAccountResponse, error) {
=======
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
>>>>>>> a4be561 (chore: rename `SubmitTx` to `SendTx` (#2255))
	ctx := sdk.UnwrapSDKContext(goCtx)

	portID, err := icatypes.NewControllerPortID(msg.Owner)
	if err != nil {
		return nil, err
	}

	channelID, err := s.registerInterchainAccount(ctx, msg.ConnectionId, portID, msg.Version)
	if err != nil {
		return nil, err
	}

	return &types.MsgRegisterAccountResponse{
		ChannelId: channelID,
	}, nil
}

<<<<<<< HEAD
// SubmitTx defines a rpc handler for MsgSubmitTx
func (k Keeper) SubmitTx(goCtx context.Context, msg *types.MsgSubmitTx) (*types.MsgSubmitTxResponse, error) {
=======
// SendTx defines a rpc handler for MsgSendTx
func (s msgServer) SendTx(goCtx context.Context, msg *types.MsgSendTx) (*types.MsgSendTxResponse, error) {
>>>>>>> a4be561 (chore: rename `SubmitTx` to `SendTx` (#2255))
	ctx := sdk.UnwrapSDKContext(goCtx)

	portID, err := icatypes.NewControllerPortID(msg.Owner)
	if err != nil {
		return nil, err
	}

	channelID, found := s.Keeper.GetActiveChannelID(ctx, msg.ConnectionId, portID)
	if !found {
		return nil, sdkerrors.Wrapf(icatypes.ErrActiveChannelNotFound, "failed to retrieve active channel for port %s", portID)
	}

	chanCap, found := s.Keeper.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(portID, channelID))
	if !found {
		return nil, sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	seq, err := s.Keeper.SendTx(ctx, chanCap, msg.ConnectionId, portID, msg.PacketData, msg.TimeoutTimestamp)
	if err != nil {
		return nil, err
	}

	return &types.MsgSendTxResponse{Sequence: seq}, nil
}
