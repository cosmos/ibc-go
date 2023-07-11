package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

const (
	// EventTypeSourceCallback is the event type for a source callback
	EventTypeSourceCallback = "ibc_src_callback"
	// EventTypeDestinationCallback is the event type for a destination callback
	EventTypeDestinationCallback = "ibc_dest_callback"

	// AttributeKeyCallbackTrigger denotes the condition that the callback is executed on:
	//   "acknowledgement": the callback is executed on the acknowledgement of the packet
	//   "timeout": the callback is executed on the timeout of the packet
	//   "recv_packet": the callback is executed on the reception of the packet
	AttributeKeyCallbackTrigger = "callback_trigger"
	// AttributeKeySourceCallbackAddress denotes the source callback contract address
	AttributeKeyCallbackAddress = "callback_address"
	// AttributeKeyCallbackResult denotes the callback result:
	//   "success": the callback is successfully executed
	//   "failure": the callback is failed to execute
	AttributeKeyCallbackResult = "callback_result"
	// AttributeKeyCallbackError denotes the callback error message
	// if no error is returned, then this key will not be included in the event
	AttributeKeyCallbackError = "callback_error"
	// AttributeKeyCallbackGasLimit denotes the custom gas limit for the callback execution
	// if custom gas limit is not in effect, then this key will not be included in the event
	AttributeKeyCallbackGasLimit = "callback_gas_limit"
	// AttributeKeyCallbackPortID denotes the port ID of the packet
	AttributeKeyCallbackSourcePortID = "callback_src_port"
	// AttributeKeyCallbackChannelID denotes the channel ID of the packet
	AttributeKeyCallbackSourceChannelID = "callback_src_channel"
	// AttributeKeyCallbackSequence denotes the sequence of the packet
	AttributeKeyCallbackSequence = "callback_sequence"
)

// EmitSourceCallbackEvent emits an event for a source callback
func EmitSourceCallbackEvent(
	ctx sdk.Context,
	packet channeltypes.Packet,
	callbackType CallbackType,
	callbackData CallbackData,
	err error,
) {
	emitCallbackEvent(ctx, packet, EventTypeSourceCallback, callbackType, callbackData, err)
}

// EmitDestinationCallbackEvent emits an event for a destination callback
func EmitDestinationCallbackEvent(
	ctx sdk.Context,
	packet channeltypes.Packet,
	callbackType CallbackType,
	callbackData CallbackData,
	err error,
) {
	emitCallbackEvent(ctx, packet, EventTypeDestinationCallback, callbackType, callbackData, err)
}

// emitCallbackEvent emits an event for a callback
func emitCallbackEvent(
	ctx sdk.Context,
	packet channeltypes.Packet,
	eventType string,
	callbackTrigger CallbackType,
	callbackData CallbackData,
	err error,
) {
	attributes := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyModule, ModuleName),
		sdk.NewAttribute(AttributeKeyCallbackTrigger, string(callbackTrigger)),
		sdk.NewAttribute(AttributeKeyCallbackAddress, callbackData.ContractAddr),
		sdk.NewAttribute(AttributeKeyCallbackGasLimit, fmt.Sprintf("%d", callbackData.GasLimit)),
		sdk.NewAttribute(AttributeKeyCallbackSourcePortID, packet.SourcePort),
		sdk.NewAttribute(AttributeKeyCallbackSourceChannelID, packet.SourceChannel),
		sdk.NewAttribute(AttributeKeyCallbackSequence, fmt.Sprintf("%d", packet.Sequence)),
	}
	if err == nil {
		attributes = append(attributes, sdk.NewAttribute(AttributeKeyCallbackResult, "success"))
	} else {
		attributes = append(
			attributes,
			sdk.NewAttribute(AttributeKeyCallbackError, err.Error()),
			sdk.NewAttribute(AttributeKeyCallbackResult, "failure"),
		)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			attributes...,
		),
	)
}
