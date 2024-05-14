package keeper

import (
	"context"

	clientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"
	channelkeeper "github.com/cosmos/ibc-go/v8/modules/core/04-channel/keeper"
	"github.com/cosmos/ibc-go/v8/modules/core/lite/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.IBCLiteKeeper = (*Keeper)(nil)

type Keeper struct {
	cdc codec.BinaryCodec

	clientKeeper  *clientkeeper.Keeper
	channelKeeper *channelkeeper.Keeper
}

// NewKeeper creates a new ibc lite Keeper. It wraps over the ibc-go client keeper and channel keeper
// to implement the required interface for the IBC lite module
func NewKeeper(cdc codec.BinaryCodec, clientKeeper *clientkeeper.Keeper, channelKeeper *channelkeeper.Keeper) *Keeper {
	return &Keeper{
		cdc:           cdc,
		clientKeeper:  clientKeeper,
		channelKeeper: channelKeeper,
	}
}

// GetCounterparty returns the counterparty channelID given the channel ID on
// the executing chain
// Note for IBC lite, the portID is not needed as there is effectively
// a single channel between the two clients that can switch between apps using the portID
func (k Keeper) GetCounterparty(goCtx context.Context, clientID string) (counterparty string) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	return k.clientKeeper.GetCounterparty(ctx, clientID)
}

func (k Keeper) SetPacketCommitment(goCtx context.Context, portID string, channelID string, sequence uint64, commitment []byte) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	k.channelKeeper.SetPacketCommitment(ctx, portID, channelID, sequence, commitment)

}

func (k Keeper) GetPacketCommitment(goCtx context.Context, portID string, channelID string, sequence uint64) []byte {
	ctx := sdk.UnwrapSDKContext(goCtx)

	return k.channelKeeper.GetPacketCommitment(ctx, portID, channelID, sequence)
}

func (k Keeper) DeletePacketCommitment(goCtx context.Context, portID string, channelID string, sequence uint64) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	k.channelKeeper.DeletePacketCommitment(ctx, portID, channelID, sequence)
}

func (k Keeper) GetNextSequenceSend(goCtx context.Context, portID, channelID string) (uint64, bool) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	return k.channelKeeper.GetNextSequenceSend(ctx, portID, channelID)
}

func (k Keeper) SetNextSequenceSend(goCtx context.Context, portID, channelID string, sequence uint64) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	k.channelKeeper.SetNextSequenceSend(ctx, portID, channelID, sequence)
}

func (k Keeper) SetPacketReceipt(goCtx context.Context, portID, channelID string, sequence uint64) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	k.channelKeeper.SetPacketReceipt(ctx, portID, channelID, sequence)
}

// SetPacketAcknowledgement writes the acknowledgement hash under the acknowledgement path
func (k Keeper) SetPacketAcknowledgement(goCtx context.Context, portID, channelID string, sequence uint64, ackHash []byte) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	k.channelKeeper.SetPacketAcknowledgement(ctx, portID, channelID, sequence, ackHash)
}
