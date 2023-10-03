package types_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (s *CallbacksTypesTestSuite) TestEvents() {
	constEvents := func() ibctesting.EventsMap {
		return ibctesting.EventsMap{
			types.EventTypeSourceCallback: {
				sdk.AttributeKeyModule:                    types.ModuleName,
				types.AttributeKeyCallbackType:            string(types.CallbackTypeAcknowledgementPacket),
				types.AttributeKeyCallbackAddress:         ibctesting.TestAccAddress,
				types.AttributeKeyCallbackGasLimit:        "100000",
				types.AttributeKeyCallbackCommitGasLimit:  "200000",
				types.AttributeKeyCallbackSourcePortID:    ibctesting.MockPort,
				types.AttributeKeyCallbackSourceChannelID: ibctesting.FirstChannelID,
				types.AttributeKeyCallbackSequence:        "1",
				types.AttributeKeyCallbackResult:          types.AttributeValueCallbackSuccess,
			},
			types.EventTypeDestinationCallback: {
				sdk.AttributeKeyModule:                   types.ModuleName,
				types.AttributeKeyCallbackType:           string(types.CallbackTypeReceivePacket),
				types.AttributeKeyCallbackAddress:        ibctesting.TestAccAddress,
				types.AttributeKeyCallbackGasLimit:       "100000",
				types.AttributeKeyCallbackCommitGasLimit: "200000",
				types.AttributeKeyCallbackDestPortID:     ibctesting.MockFeePort,
				types.AttributeKeyCallbackDestChannelID:  ibctesting.InvalidID,
				types.AttributeKeyCallbackSequence:       "1",
				types.AttributeKeyCallbackResult:         types.AttributeValueCallbackSuccess,
			},
		}
	}

	var expEvents ibctesting.EventsMap
	testCases := []struct {
		name          string
		packet        channeltypes.Packet
		callbackType  types.CallbackType
		callbackData  types.CallbackData
		callbackError error
		malleate      func()
	}{
		{
			"success: ack callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeAcknowledgementPacket,
			types.CallbackData{
				CallbackAddress:   ibctesting.TestAccAddress,
				ExecutionGasLimit: 100_000,
				CommitGasLimit:    200_000,
			},
			nil,
			func() {},
		},
		{
			"success: send packet callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeSendPacket,
			types.CallbackData{
				CallbackAddress:   ibctesting.TestAccAddress,
				ExecutionGasLimit: 100_000,
				CommitGasLimit:    200_000,
			},
			nil,
			func() {
				expEvents[types.EventTypeSourceCallback][types.AttributeKeyCallbackType] = string(types.CallbackTypeSendPacket)
			},
		},
		{
			"success: timeout callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeTimeoutPacket,
			types.CallbackData{
				CallbackAddress:   ibctesting.TestAccAddress,
				ExecutionGasLimit: 100_000,
				CommitGasLimit:    200_000,
			},
			nil,
			func() {
				expEvents[types.EventTypeSourceCallback][types.AttributeKeyCallbackType] = string(types.CallbackTypeTimeoutPacket)
			},
		},
		{
			"success: receive packet callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeReceivePacket,
			types.CallbackData{
				CallbackAddress:   ibctesting.TestAccAddress,
				ExecutionGasLimit: 100_000,
				CommitGasLimit:    200_000,
			},
			nil,
			func() {},
		},
		{
			"success: unknown callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			"something",
			types.CallbackData{
				CallbackAddress:   ibctesting.TestAccAddress,
				ExecutionGasLimit: 100_000,
				CommitGasLimit:    200_000,
			},
			nil,
			func() {
				expEvents[types.EventTypeSourceCallback][types.AttributeKeyCallbackType] = "something"
			},
		},
		{
			"failure: ack callback with error",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeAcknowledgementPacket,
			types.CallbackData{
				CallbackAddress:   ibctesting.TestAccAddress,
				ExecutionGasLimit: 100_000,
				CommitGasLimit:    200_000,
			},
			types.ErrNotPacketDataProvider,
			func() {
				expEvents[types.EventTypeSourceCallback][types.AttributeKeyCallbackResult] = types.AttributeValueCallbackFailure
				expEvents[types.EventTypeSourceCallback][types.AttributeKeyCallbackError] = types.ErrNotPacketDataProvider.Error()
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			expEvents = constEvents()
			tc.malleate()
			newCtx := sdk.Context{}.WithEventManager(sdk.NewEventManager())

			switch tc.callbackType {
			case types.CallbackTypeReceivePacket:
				delete(expEvents, types.EventTypeSourceCallback)
				types.EmitCallbackEvent(
					newCtx, tc.packet.GetDestPort(), tc.packet.GetDestChannel(),
					tc.packet.GetSequence(), tc.callbackType, tc.callbackData, tc.callbackError,
				)

			default:
				delete(expEvents, types.EventTypeDestinationCallback)
				types.EmitCallbackEvent(
					newCtx, tc.packet.GetSourcePort(), tc.packet.GetSourceChannel(),
					tc.packet.GetSequence(), tc.callbackType, tc.callbackData, tc.callbackError,
				)
			}

			events := newCtx.EventManager().Events().ToABCIEvents()
			ibctesting.AssertEventsLegacy(&s.Suite, expEvents, events)
		})
	}
}
