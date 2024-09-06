package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/event"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// emitIncentivizedPacketEvent emits an event containing information on the total amount of fees incentivizing
// a specific packet. It should be emitted on every fee escrowed for the given packetID.
func emitIncentivizedPacketEvent(ctx context.Context, env appmodule.Environment, packetID channeltypes.PacketId, packetFees types.PacketFees) {
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

	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeIncentivizedPacket,
		event.Attribute{Key: channeltypes.AttributeKeyPortID, Value: packetID.PortId},
		event.Attribute{Key: channeltypes.AttributeKeyChannelID, Value: packetID.ChannelId},
		event.Attribute{Key: channeltypes.AttributeKeySequence, Value: fmt.Sprint(packetID.Sequence)},
		event.Attribute{Key: types.AttributeKeyRecvFee, Value: totalRecvFees.String()},
		event.Attribute{Key: types.AttributeKeyAckFee, Value: totalAckFees.String()},
		event.Attribute{Key: types.AttributeKeyTimeoutFee, Value: totalTimeoutFees.String()},
	)

	env.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.ModuleName},
	)

}

// emitRegisterPayeeEvent emits an event containing information of a registered payee for a relayer on a particular channel
func emitRegisterPayeeEvent(ctx context.Context, env appmodule.Environment, relayer, payee, channelID string) {
	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeRegisterPayee,
		event.Attribute{Key: types.AttributeKeyRelayer, Value: relayer},
		event.Attribute{Key: types.AttributeKeyPayee, Value: payee},
		event.Attribute{Key: types.AttributeKeyChannelID, Value: channelID},
	)
	env.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.ModuleName},
	)
}

// emitRegisterCounterpartyPayeeEvent emits an event containing information of a registered counterparty payee for a relayer on a particular channel
func emitRegisterCounterpartyPayeeEvent(ctx context.Context, env appmodule.Environment, relayer, counterpartyPayee, channelID string) {
	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeRegisterCounterpartyPayee,
		event.Attribute{Key: types.AttributeKeyRelayer, Value: relayer},
		event.Attribute{Key: types.AttributeKeyCounterpartyPayee, Value: counterpartyPayee},
		event.Attribute{Key: types.AttributeKeyChannelID, Value: channelID},
	)
	env.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.ModuleName},
	)
}

// emitDistributeFeeEvent emits an event containing a distribution fee and receiver address
func emitDistributeFeeEvent(ctx context.Context, env appmodule.Environment, receiver string, fee sdk.Coins) {
	env.EventService.EventManager(ctx).EmitKV(
		types.EventTypeDistributeFee,
		event.Attribute{Key: types.AttributeKeyReceiver, Value: receiver},
		event.Attribute{Key: types.AttributeKeyFee, Value: fee.String()},
	)
	env.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.Attribute{Key: sdk.AttributeKeyModule, Value: types.ModuleName},
	)
}
