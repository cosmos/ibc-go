package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

var _ types.MsgServer = Keeper{}

// RegisterPayee defines a rpc handler method for MsgRegisterPayee
// RegisterPayee is called by the relayer on each channelEnd and allows them to set an optional
// payee to which escrowed packet fees will be paid out. The payee should be registered on the source chain from which
// packets originate as this is where fee distribution takes place. This function may be called more than once by a relayer,
// in which case, the latest payee is always used.
func (k Keeper) RegisterPayee(goCtx context.Context, msg *types.MsgRegisterPayee) (*types.MsgRegisterPayeeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// only register payee address if the channel exists and is fee enabled
	if _, found := k.channelKeeper.GetChannel(ctx, msg.PortId, msg.ChannelId); !found {
		return nil, channeltypes.ErrChannelNotFound
	}

	if !k.IsFeeEnabled(ctx, msg.PortId, msg.ChannelId) {
		return nil, types.ErrFeeNotEnabled
	}

	k.SetPayeeAddress(ctx, msg.RelayerAddress, msg.Payee, msg.ChannelId)

	k.Logger(ctx).Info("registering payee address for relayer", "relayer address", msg.RelayerAddress, "payee address", msg.Payee, "channel", msg.ChannelId)

	return &types.MsgRegisterPayeeResponse{}, nil
}

// RegisterCounterpartyAddress is called by the relayer on each channelEnd and allows them to specify their counterparty address before relaying
// This ensures they will be properly compensated for forward relaying on the source chain since the destination chain must send back relayer's source address (counterparty address) in acknowledgement
// This function may be called more than once by relayers, in which case, the previous counterparty address will be overwritten by the new counterparty address
func (k Keeper) RegisterCounterpartyAddress(goCtx context.Context, msg *types.MsgRegisterCounterpartyAddress) (*types.MsgRegisterCounterpartyAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// only register counterparty address if the channel exists and is fee enabled
	if _, found := k.channelKeeper.GetChannel(ctx, msg.PortId, msg.ChannelId); !found {
		return nil, channeltypes.ErrChannelNotFound
	}

	if !k.IsFeeEnabled(ctx, msg.PortId, msg.ChannelId) {
		return nil, types.ErrFeeNotEnabled
	}

	k.SetCounterpartyAddress(ctx, msg.Address, msg.CounterpartyAddress, msg.ChannelId)

	k.Logger(ctx).Info("registering counterparty address for relayer", "address", msg.Address, "counterparty address", msg.CounterpartyAddress, "channel", msg.ChannelId)

	return &types.MsgRegisterCounterpartyAddressResponse{}, nil
}

// PayPacketFee defines a rpc handler method for MsgPayPacketFee
// PayPacketFee is an open callback that may be called by any module/user that wishes to escrow funds in order to relay the packet with the next sequence
func (k Keeper) PayPacketFee(goCtx context.Context, msg *types.MsgPayPacketFee) (*types.MsgPayPacketFeeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsFeeEnabled(ctx, msg.SourcePortId, msg.SourceChannelId) {
		// users may not escrow fees on this channel. Must send packets without a fee message
		return nil, types.ErrFeeNotEnabled
	}

	if k.IsLocked(ctx) {
		return nil, types.ErrFeeModuleLocked
	}

	// get the next sequence
	sequence, found := k.GetNextSequenceSend(ctx, msg.SourcePortId, msg.SourceChannelId)
	if !found {
		return nil, channeltypes.ErrSequenceSendNotFound
	}

	packetID := channeltypes.NewPacketId(msg.SourcePortId, msg.SourceChannelId, sequence)
	packetFee := types.NewPacketFee(msg.Fee, msg.Signer, msg.Relayers)

	if err := k.escrowPacketFee(ctx, packetID, packetFee); err != nil {
		return nil, err
	}

	return &types.MsgPayPacketFeeResponse{}, nil
}

// PayPacketFee defines a rpc handler method for MsgPayPacketFee
// PayPacketFee is an open callback that may be called by any module/user that wishes to escrow funds in order to
// incentivize the relaying of a known packet. Only packets which have been sent and have not gone through the
// packet life cycle may be incentivized.
func (k Keeper) PayPacketFeeAsync(goCtx context.Context, msg *types.MsgPayPacketFeeAsync) (*types.MsgPayPacketFeeAsyncResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsFeeEnabled(ctx, msg.PacketId.PortId, msg.PacketId.ChannelId) {
		// users may not escrow fees on this channel. Must send packets without a fee message
		return nil, types.ErrFeeNotEnabled
	}

	if k.IsLocked(ctx) {
		return nil, types.ErrFeeModuleLocked
	}

	nextSeqSend, found := k.GetNextSequenceSend(ctx, msg.PacketId.PortId, msg.PacketId.ChannelId)
	if !found {
		return nil, sdkerrors.Wrapf(channeltypes.ErrSequenceSendNotFound, "channel does not exist, portID: %s, channelID: %s", msg.PacketId.PortId, msg.PacketId.ChannelId)
	}

	// only allow incentivizing of packets which have been sent
	if msg.PacketId.Sequence >= nextSeqSend {
		return nil, channeltypes.ErrPacketNotSent
	}

	// only allow incentivizng of packets which have not completed the packet life cycle
	if bz := k.GetPacketCommitment(ctx, msg.PacketId.PortId, msg.PacketId.ChannelId, msg.PacketId.Sequence); len(bz) == 0 {
		return nil, sdkerrors.Wrapf(channeltypes.ErrPacketCommitmentNotFound, "packet has already been acknowledged or timed out")
	}

	if err := k.escrowPacketFee(ctx, msg.PacketId, msg.PacketFee); err != nil {
		return nil, err
	}

	return &types.MsgPayPacketFeeAsyncResponse{}, nil
}
