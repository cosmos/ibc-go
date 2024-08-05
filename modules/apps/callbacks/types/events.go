package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// EventTypeSourceCallback is the event type for a source callback
	EventTypeSourceCallback = "ibc_src_callback"
	// EventTypeDestinationCallback is the event type for a destination callback
	EventTypeDestinationCallback = "ibc_dest_callback"

	// AttributeKeyCallbackType denotes the condition that the callback is executed on:
	//   "acknowledgement": the callback is executed on the acknowledgement of the packet
	//   "timeout": the callback is executed on the timeout of the packet
	//   "recv_packet": the callback is executed on the reception of the packet
	AttributeKeyCallbackType = "callback_type"
	// AttributeKeyCallbackAddress denotes the callback address
	AttributeKeyCallbackAddress = "callback_address"
	// AttributeKeyCallbackResult denotes the callback result:
	//   AttributeValueCallbackSuccess: the callback is successfully executed
	//   AttributeValueCallbackFailure: the callback has failed to execute
	AttributeKeyCallbackResult = "callback_result"
	// AttributeKeyCallbackError denotes the callback error message
	// if no error is returned, then this key will not be included in the event
	AttributeKeyCallbackError = "callback_error"
	// AttributeKeyCallbackGasLimit denotes the custom gas limit for the callback execution
	// if custom gas limit is not in effect, then this key will not be included in the event
	AttributeKeyCallbackGasLimit = "callback_exec_gas_limit"
	// AttributeKeyCallbackCommitGasLimit denotes the gas needed to commit the callback even
	// if the callback execution fails due to out of gas.
	AttributeKeyCallbackCommitGasLimit = "callback_commit_gas_limit"
	// AttributeKeyCallbackSourcePortID denotes the source port ID of the packet
	AttributeKeyCallbackSourcePortID = "packet_src_port"
	// AttributeKeyCallbackSourceChannelID denotes the source channel ID of the packet
	AttributeKeyCallbackSourceChannelID = "packet_src_channel"
	// AttributeKeyCallbackDestPortID denotes the destination port ID of the packet
	AttributeKeyCallbackDestPortID = "packet_dest_port"
	// AttributeKeyCallbackDestChannelID denotes the destination channel ID of the packet
	AttributeKeyCallbackDestChannelID = "packet_dest_channel"
	// AttributeKeyCallbackSequence denotes the sequence of the packet
	AttributeKeyCallbackSequence = "packet_sequence"
	// AttributeKeyCallbackBaseApplicationVersion denotes the callback base application version
	AttributeKeyCallbackBaseApplicationVersion = "callback_base_application_version"
	// AttributeValueCallbackSuccess denotes that the callback is successfully executed
	AttributeValueCallbackSuccess = "success"
	// AttributeValueCallbackFailure denotes that the callback has failed to execute
	AttributeValueCallbackFailure = "failure"
)

// EmitCallbackEvent emits an event for a callback
func EmitCallbackEvent(
	ctx sdk.Context,
	portID,
	channelID string,
	sequence uint64,
	callbackType CallbackType,
	callbackData CallbackData,
	err error,
) {
	attributes := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyModule, ModuleName),
		sdk.NewAttribute(AttributeKeyCallbackType, string(callbackType)),
		sdk.NewAttribute(AttributeKeyCallbackAddress, callbackData.CallbackAddress),
		sdk.NewAttribute(AttributeKeyCallbackGasLimit, fmt.Sprintf("%d", callbackData.ExecutionGasLimit)),
		sdk.NewAttribute(AttributeKeyCallbackCommitGasLimit, fmt.Sprintf("%d", callbackData.CommitGasLimit)),
		sdk.NewAttribute(AttributeKeyCallbackSequence, fmt.Sprintf("%d", sequence)),
		sdk.NewAttribute(AttributeKeyCallbackBaseApplicationVersion, callbackData.ApplicationVersion),
	}
	if err == nil {
		attributes = append(attributes, sdk.NewAttribute(AttributeKeyCallbackResult, AttributeValueCallbackSuccess))
	} else {
		attributes = append(
			attributes,
			sdk.NewAttribute(AttributeKeyCallbackError, err.Error()),
			sdk.NewAttribute(AttributeKeyCallbackResult, AttributeValueCallbackFailure),
		)
	}

	var eventType string
	switch callbackType {
	case CallbackTypeReceivePacket:
		eventType = EventTypeDestinationCallback
		attributes = append(
			attributes, sdk.NewAttribute(AttributeKeyCallbackDestPortID, portID),
			sdk.NewAttribute(AttributeKeyCallbackDestChannelID, channelID),
		)
	default:
		eventType = EventTypeSourceCallback
		attributes = append(
			attributes, sdk.NewAttribute(AttributeKeyCallbackSourcePortID, portID),
			sdk.NewAttribute(AttributeKeyCallbackSourceChannelID, channelID),
		)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			attributes...,
		),
	)
}
