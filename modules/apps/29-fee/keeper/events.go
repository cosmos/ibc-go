package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/event"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// emitIncentivizedPacketEvent emits an event containing information on the total amount of fees incentivizing
// a specific packet. It should be emitted on every fee escrowed for the given packetID.
func (k Keeper) emitIncentivizedPacketEvent(ctx context.Context, packetID channeltypes.PacketId, packetFees types.PacketFees) error {
	var (
		totalRecvFees    sdk.Coins
		totalAckFees     sdk.Coins
		totalTimeoutFees sdk.Coins
	)

	for _, fee := range packetFees.PacketFees {
		// only emit total fees for packet fees which allow any relayer to relay
		if fee.Relayers == nil {
			totalRecvFees = totalRecvFees.Add(fee.Fee.RecvFee...)
			totalAckFees = totalAckFees.Add(fee.Fee.AckFee...)
			totalTimeoutFees = totalTimeoutFees.Add(fee.Fee.TimeoutFee...)
		}
	}

	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeIncentivizedPacket,
		event.NewAttribute(channeltypes.AttributeKeyPortID, packetID.PortId),
		event.NewAttribute(channeltypes.AttributeKeyChannelID, packetID.ChannelId),
		event.NewAttribute(channeltypes.AttributeKeySequence, fmt.Sprint(packetID.Sequence)),
		event.NewAttribute(types.AttributeKeyRecvFee, totalRecvFees.String()),
		event.NewAttribute(types.AttributeKeyAckFee, totalAckFees.String()),
		event.NewAttribute(types.AttributeKeyTimeoutFee, totalTimeoutFees.String()),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	)
}

// emitRegisterPayeeEvent emits an event containing information of a registered payee for a relayer on a particular channel
func (k Keeper) emitRegisterPayeeEvent(ctx context.Context, relayer, payee, channelID string) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeRegisterPayee,
		event.NewAttribute(types.AttributeKeyRelayer, relayer),
		event.NewAttribute(types.AttributeKeyPayee, payee),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	)
}

// emitRegisterCounterpartyPayeeEvent emits an event containing information of a registered counterparty payee for a relayer on a particular channel
func (k Keeper) emitRegisterCounterpartyPayeeEvent(ctx context.Context, relayer, counterpartyPayee, channelID string) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeRegisterCounterpartyPayee,
		event.NewAttribute(types.AttributeKeyRelayer, relayer),
		event.NewAttribute(types.AttributeKeyCounterpartyPayee, counterpartyPayee),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	)
}

// emitDistributeFeeEvent emits an event containing a distribution fee and receiver address
func (k Keeper) emitDistributeFeeEvent(ctx context.Context, receiver string, fee sdk.Coins) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeDistributeFee,
		event.NewAttribute(types.AttributeKeyReceiver, receiver),
		event.NewAttribute(types.AttributeKeyFee, fee.String()),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	)
}
