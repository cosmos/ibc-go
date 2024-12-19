package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"cosmossdk.io/core/event"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// emitChannelOpenInitEvent emits a channel open init event
func (k *Keeper) emitChannelOpenInitEvent(ctx context.Context, portID string, channelID string, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelOpenInit,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
		event.NewAttribute(types.AttributeKeyVersion, channel.Version),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitChannelOpenTryEvent emits a channel open try event
func (k *Keeper) emitChannelOpenTryEvent(ctx context.Context, portID string, channelID string, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelOpenTry,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
		event.NewAttribute(types.AttributeKeyVersion, channel.Version),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitChannelOpenAckEvent emits a channel open acknowledge event
func (k *Keeper) emitChannelOpenAckEvent(ctx context.Context, portID string, channelID string, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelOpenAck,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitChannelOpenConfirmEvent emits a channel open confirm event
func (k *Keeper) emitChannelOpenConfirmEvent(ctx context.Context, portID string, channelID string, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelOpenConfirm,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitChannelCloseInitEvent emits a channel close init event
func (k *Keeper) emitChannelCloseInitEvent(ctx context.Context, portID string, channelID string, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelCloseInit,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitChannelCloseConfirmEvent emits a channel close confirm event
func (k *Keeper) emitChannelCloseConfirmEvent(ctx context.Context, portID string, channelID string, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelCloseConfirm,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitSendPacketEvent emits an event with packet data along with other packet information for relayer
// to pick up and relay to other chain
func (k *Keeper) emitSendPacketEvent(ctx context.Context, packet types.Packet, channel types.Channel, timeoutHeight exported.Height) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeSendPacket,
		event.NewAttribute(types.AttributeKeyDataHex, hex.EncodeToString(packet.GetData())),
		event.NewAttribute(types.AttributeKeyTimeoutHeight, timeoutHeight.String()),
		event.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.GetTimeoutTimestamp())),
		event.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.GetSequence())),
		event.NewAttribute(types.AttributeKeySrcPort, packet.GetSourcePort()),
		event.NewAttribute(types.AttributeKeySrcChannel, packet.GetSourceChannel()),
		event.NewAttribute(types.AttributeKeyDstPort, packet.GetDestPort()),
		event.NewAttribute(types.AttributeKeyDstChannel, packet.GetDestChannel()),
		event.NewAttribute(types.AttributeKeyChannelOrdering, channel.Ordering.String()),
		// we only support 1-hop packets now, and that is the most important hop for a relayer
		// (is it going to a chain I am connected to)
		event.NewAttribute(types.AttributeKeyConnection, channel.ConnectionHops[0]), // DEPRECATED
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitRecvPacketEvent emits a receive packet event. It will be emitted both the first time a packet
// is received for a certain sequence and for all duplicate receives.
func (k *Keeper) emitRecvPacketEvent(ctx context.Context, packet types.Packet, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeRecvPacket,
		event.NewAttribute(types.AttributeKeyDataHex, hex.EncodeToString(packet.GetData())),
		event.NewAttribute(types.AttributeKeyTimeoutHeight, packet.GetTimeoutHeight().String()),
		event.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.GetTimeoutTimestamp())),
		event.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.GetSequence())),
		event.NewAttribute(types.AttributeKeySrcPort, packet.GetSourcePort()),
		event.NewAttribute(types.AttributeKeySrcChannel, packet.GetSourceChannel()),
		event.NewAttribute(types.AttributeKeyDstPort, packet.GetDestPort()),
		event.NewAttribute(types.AttributeKeyDstChannel, packet.GetDestChannel()),
		event.NewAttribute(types.AttributeKeyChannelOrdering, channel.Ordering.String()),
		// we only support 1-hop packets now, and that is the most important hop for a relayer
		// (is it going to a chain I am connected to)
		event.NewAttribute(types.AttributeKeyConnection, channel.ConnectionHops[0]), // DEPRECATED
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitWriteAcknowledgementEvent emits an event that the relayer can query for
func (k *Keeper) emitWriteAcknowledgementEvent(ctx context.Context, packet types.Packet, channel types.Channel, acknowledgement []byte) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeWriteAck,
		event.NewAttribute(types.AttributeKeyDataHex, hex.EncodeToString(packet.GetData())),
		event.NewAttribute(types.AttributeKeyTimeoutHeight, packet.GetTimeoutHeight().String()),
		event.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.GetTimeoutTimestamp())),
		event.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.GetSequence())),
		event.NewAttribute(types.AttributeKeySrcPort, packet.GetSourcePort()),
		event.NewAttribute(types.AttributeKeySrcChannel, packet.GetSourceChannel()),
		event.NewAttribute(types.AttributeKeyDstPort, packet.GetDestPort()),
		event.NewAttribute(types.AttributeKeyDstChannel, packet.GetDestChannel()),
		event.NewAttribute(types.AttributeKeyAckHex, hex.EncodeToString(acknowledgement)),
		event.NewAttribute(types.AttributeKeyChannelOrdering, channel.Ordering.String()),
		// we only support 1-hop packets now, and that is the most important hop for a relayer
		// (is it going to a chain I am connected to)
		event.NewAttribute(types.AttributeKeyConnection, channel.ConnectionHops[0]), // DEPRECATED
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitAcknowledgePacketEvent emits an acknowledge packet event. It will be emitted both the first time
// a packet is acknowledged for a certain sequence and for all duplicate acknowledgements.
func (k *Keeper) emitAcknowledgePacketEvent(ctx context.Context, packet types.Packet, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeAcknowledgePacket,
		event.NewAttribute(types.AttributeKeyTimeoutHeight, packet.GetTimeoutHeight().String()),
		event.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.GetTimeoutTimestamp())),
		event.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.GetSequence())),
		event.NewAttribute(types.AttributeKeySrcPort, packet.GetSourcePort()),
		event.NewAttribute(types.AttributeKeySrcChannel, packet.GetSourceChannel()),
		event.NewAttribute(types.AttributeKeyDstPort, packet.GetDestPort()),
		event.NewAttribute(types.AttributeKeyDstChannel, packet.GetDestChannel()),
		event.NewAttribute(types.AttributeKeyChannelOrdering, channel.Ordering.String()),
		// we only support 1-hop packets now, and that is the most important hop for a relayer
		// (is it going to a chain I am connected to)
		event.NewAttribute(types.AttributeKeyConnection, channel.ConnectionHops[0]), // DEPRECATED
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitTimeoutPacketEvent emits a timeout packet event. It will be emitted both the first time a packet
// is timed out for a certain sequence and for all duplicate timeouts.
func (k *Keeper) emitTimeoutPacketEvent(ctx context.Context, packet types.Packet, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeTimeoutPacket,
		event.NewAttribute(types.AttributeKeyTimeoutHeight, packet.GetTimeoutHeight().String()),
		event.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", packet.GetTimeoutTimestamp())),
		event.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.GetSequence())),
		event.NewAttribute(types.AttributeKeySrcPort, packet.GetSourcePort()),
		event.NewAttribute(types.AttributeKeySrcChannel, packet.GetSourceChannel()),
		event.NewAttribute(types.AttributeKeyDstPort, packet.GetDestPort()),
		event.NewAttribute(types.AttributeKeyDstChannel, packet.GetDestChannel()),
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
		event.NewAttribute(types.AttributeKeyChannelOrdering, channel.Ordering.String()),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitChannelClosedEvent emits a channel closed event.
func (k *Keeper) emitChannelClosedEvent(ctx context.Context, packet types.Packet, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelClosed,
		event.NewAttribute(types.AttributeKeyPortID, packet.GetSourcePort()),
		event.NewAttribute(types.AttributeKeyChannelID, packet.GetSourceChannel()),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyConnectionID, channel.ConnectionHops[0]),
		event.NewAttribute(types.AttributeKeyChannelOrdering, channel.Ordering.String()),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// EmitChannelUpgradeInitEvent emits a channel upgrade init event
func (k *Keeper) EmitChannelUpgradeInitEvent(ctx context.Context, portID string, channelID string, channel types.Channel, upgrade types.Upgrade) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelUpgradeInit,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// EmitChannelUpgradeTryEvent emits a channel upgrade try event
func (k *Keeper) EmitChannelUpgradeTryEvent(ctx context.Context, portID string, channelID string, channel types.Channel, upgrade types.Upgrade) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelUpgradeTry,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// EmitChannelUpgradeAckEvent emits a channel upgrade ack event
func (k *Keeper) EmitChannelUpgradeAckEvent(ctx context.Context, portID string, channelID string, channel types.Channel, upgrade types.Upgrade) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelUpgradeAck,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// EmitChannelUpgradeConfirmEvent emits a channel upgrade confirm event
func (k *Keeper) EmitChannelUpgradeConfirmEvent(ctx context.Context, portID, channelID string, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelUpgradeConfirm,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeKeyChannelState, channel.State.String()),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// EmitChannelUpgradeOpenEvent emits a channel upgrade open event
func (k *Keeper) EmitChannelUpgradeOpenEvent(ctx context.Context, portID string, channelID string, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelUpgradeOpen,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeKeyChannelState, channel.State.String()),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// EmitChannelUpgradeTimeoutEvent emits an upgrade timeout event.
func (k *Keeper) EmitChannelUpgradeTimeoutEvent(ctx context.Context, portID string, channelID string, channel types.Channel, upgrade types.Upgrade) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelUpgradeTimeout,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyUpgradeTimeoutHeight, upgrade.Timeout.Height.String()),
		event.NewAttribute(types.AttributeKeyUpgradeTimeoutTimestamp, fmt.Sprintf("%d", upgrade.Timeout.Timestamp)),
		event.NewAttribute(types.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// EmitErrorReceiptEvent emits an error receipt event
func (k *Keeper) EmitErrorReceiptEvent(ctx context.Context, portID string, channelID string, channel types.Channel, err error) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelUpgradeError,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
		event.NewAttribute(types.AttributeKeyErrorReceipt, err.Error()),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// EmitChannelUpgradeCancelEvent emits an upgraded cancelled event.
func (k *Keeper) EmitChannelUpgradeCancelEvent(ctx context.Context, portID string, channelID string, channel types.Channel, upgrade types.Upgrade) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelUpgradeCancel,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyUpgradeSequence, fmt.Sprintf("%d", channel.UpgradeSequence)),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}

// emitChannelFlushCompleteEvent emits an flushing event.
func (k *Keeper) emitChannelFlushCompleteEvent(ctx context.Context, portID string, channelID string, channel types.Channel) error {
	if err := k.EventService.EventManager(ctx).EmitKV(
		types.EventTypeChannelFlushComplete,
		event.NewAttribute(types.AttributeKeyPortID, portID),
		event.NewAttribute(types.AttributeKeyChannelID, channelID),
		event.NewAttribute(types.AttributeCounterpartyPortID, channel.Counterparty.PortId),
		event.NewAttribute(types.AttributeCounterpartyChannelID, channel.Counterparty.ChannelId),
		event.NewAttribute(types.AttributeKeyChannelState, channel.State.String()),
	); err != nil {
		return err
	}

	return k.EventService.EventManager(ctx).EmitKV(
		sdk.EventTypeMessage,
		event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
	)
}
