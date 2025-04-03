package types_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *CallbacksTypesTestSuite) TestEvents() {
	testCases := []struct {
		name           string
		packet         channeltypes.Packet
		callbackType   types.CallbackType
		callbackData   types.CallbackData
		callbackError  error
		expectedEvents func() []abci.Event
	}{
		{
			"success: ack callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				transfertypes.PortID, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeAcknowledgementPacket,
			types.CallbackData{
				CallbackAddress:    ibctesting.TestAccAddress,
				ExecutionGasLimit:  100_000,
				CommitGasLimit:     200_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
			func() []abci.Event {
				return sdk.Events{
					sdk.NewEvent(
						types.EventTypeSourceCallback,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
						sdk.NewAttribute(types.AttributeKeyCallbackType, string(types.CallbackTypeAcknowledgementPacket)),
						sdk.NewAttribute(types.AttributeKeyCallbackAddress, ibctesting.TestAccAddress),
						sdk.NewAttribute(types.AttributeKeyCallbackGasLimit, "100000"),
						sdk.NewAttribute(types.AttributeKeyCallbackCommitGasLimit, "200000"),
						sdk.NewAttribute(types.AttributeKeyCallbackSourcePortID, ibctesting.MockPort),
						sdk.NewAttribute(types.AttributeKeyCallbackSourceChannelID, ibctesting.FirstChannelID),
						sdk.NewAttribute(types.AttributeKeyCallbackSequence, "1"),
						sdk.NewAttribute(types.AttributeKeyCallbackResult, types.AttributeValueCallbackSuccess),
						sdk.NewAttribute(types.AttributeKeyCallbackBaseApplicationVersion, transfertypes.V1),
					),
				}.ToABCIEvents()
			},
		},
		{
			"success: send packet callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				transfertypes.PortID, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeSendPacket,
			types.CallbackData{
				CallbackAddress:    ibctesting.TestAccAddress,
				ExecutionGasLimit:  100_000,
				CommitGasLimit:     200_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
			func() []abci.Event {
				return sdk.Events{
					sdk.NewEvent(
						types.EventTypeSourceCallback,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
						sdk.NewAttribute(types.AttributeKeyCallbackType, string(types.CallbackTypeSendPacket)),
						sdk.NewAttribute(types.AttributeKeyCallbackAddress, ibctesting.TestAccAddress),
						sdk.NewAttribute(types.AttributeKeyCallbackGasLimit, "100000"),
						sdk.NewAttribute(types.AttributeKeyCallbackCommitGasLimit, "200000"),
						sdk.NewAttribute(types.AttributeKeyCallbackSourcePortID, ibctesting.MockPort),
						sdk.NewAttribute(types.AttributeKeyCallbackSourceChannelID, ibctesting.FirstChannelID),
						sdk.NewAttribute(types.AttributeKeyCallbackSequence, "1"),
						sdk.NewAttribute(types.AttributeKeyCallbackResult, types.AttributeValueCallbackSuccess),
						sdk.NewAttribute(types.AttributeKeyCallbackBaseApplicationVersion, transfertypes.V1),
					),
				}.ToABCIEvents()
			},
		},
		{
			"success: timeout callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				transfertypes.PortID, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeTimeoutPacket,
			types.CallbackData{
				CallbackAddress:    ibctesting.TestAccAddress,
				ExecutionGasLimit:  100_000,
				CommitGasLimit:     200_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
			func() []abci.Event {
				return sdk.Events{
					sdk.NewEvent(
						types.EventTypeSourceCallback,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
						sdk.NewAttribute(types.AttributeKeyCallbackType, string(types.CallbackTypeTimeoutPacket)),
						sdk.NewAttribute(types.AttributeKeyCallbackAddress, ibctesting.TestAccAddress),
						sdk.NewAttribute(types.AttributeKeyCallbackGasLimit, "100000"),
						sdk.NewAttribute(types.AttributeKeyCallbackCommitGasLimit, "200000"),
						sdk.NewAttribute(types.AttributeKeyCallbackSourcePortID, ibctesting.MockPort),
						sdk.NewAttribute(types.AttributeKeyCallbackSourceChannelID, ibctesting.FirstChannelID),
						sdk.NewAttribute(types.AttributeKeyCallbackSequence, "1"),
						sdk.NewAttribute(types.AttributeKeyCallbackResult, types.AttributeValueCallbackSuccess),
						sdk.NewAttribute(types.AttributeKeyCallbackBaseApplicationVersion, transfertypes.V1),
					),
				}.ToABCIEvents()
			},
		},
		{
			"success: receive packet callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				transfertypes.PortID, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeReceivePacket,
			types.CallbackData{
				CallbackAddress:    ibctesting.TestAccAddress,
				ExecutionGasLimit:  100_000,
				CommitGasLimit:     200_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
			func() []abci.Event {
				return sdk.Events{
					sdk.NewEvent(
						types.EventTypeDestinationCallback,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
						sdk.NewAttribute(types.AttributeKeyCallbackType, string(types.CallbackTypeReceivePacket)),
						sdk.NewAttribute(types.AttributeKeyCallbackAddress, ibctesting.TestAccAddress),
						sdk.NewAttribute(types.AttributeKeyCallbackGasLimit, "100000"),
						sdk.NewAttribute(types.AttributeKeyCallbackCommitGasLimit, "200000"),
						sdk.NewAttribute(types.AttributeKeyCallbackDestPortID, transfertypes.PortID),
						sdk.NewAttribute(types.AttributeKeyCallbackDestChannelID, ibctesting.InvalidID),
						sdk.NewAttribute(types.AttributeKeyCallbackSequence, "1"),
						sdk.NewAttribute(types.AttributeKeyCallbackResult, types.AttributeValueCallbackSuccess),
						sdk.NewAttribute(types.AttributeKeyCallbackBaseApplicationVersion, transfertypes.V1),
					),
				}.ToABCIEvents()
			},
		},
		{
			"success: unknown callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				transfertypes.PortID, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			"something",
			types.CallbackData{
				CallbackAddress:    ibctesting.TestAccAddress,
				ExecutionGasLimit:  100_000,
				CommitGasLimit:     200_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
			func() []abci.Event {
				return sdk.Events{
					sdk.NewEvent(
						types.EventTypeSourceCallback,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
						sdk.NewAttribute(types.AttributeKeyCallbackType, "something"),
						sdk.NewAttribute(types.AttributeKeyCallbackAddress, ibctesting.TestAccAddress),
						sdk.NewAttribute(types.AttributeKeyCallbackGasLimit, "100000"),
						sdk.NewAttribute(types.AttributeKeyCallbackCommitGasLimit, "200000"),
						sdk.NewAttribute(types.AttributeKeyCallbackSourcePortID, ibctesting.MockPort),
						sdk.NewAttribute(types.AttributeKeyCallbackSourceChannelID, ibctesting.FirstChannelID),
						sdk.NewAttribute(types.AttributeKeyCallbackSequence, "1"),
						sdk.NewAttribute(types.AttributeKeyCallbackResult, types.AttributeValueCallbackSuccess),
						sdk.NewAttribute(types.AttributeKeyCallbackBaseApplicationVersion, transfertypes.V1),
					),
				}.ToABCIEvents()
			},
		},
		{
			"failure: ack callback with error",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				transfertypes.PortID, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeAcknowledgementPacket,
			types.CallbackData{
				CallbackAddress:    ibctesting.TestAccAddress,
				ExecutionGasLimit:  100_000,
				CommitGasLimit:     200_000,
				ApplicationVersion: transfertypes.V1,
			},
			types.ErrNotPacketDataProvider,
			func() []abci.Event {
				return sdk.Events{
					sdk.NewEvent(
						types.EventTypeSourceCallback,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
						sdk.NewAttribute(types.AttributeKeyCallbackType, string(types.CallbackTypeAcknowledgementPacket)),
						sdk.NewAttribute(types.AttributeKeyCallbackAddress, ibctesting.TestAccAddress),
						sdk.NewAttribute(types.AttributeKeyCallbackGasLimit, "100000"),
						sdk.NewAttribute(types.AttributeKeyCallbackCommitGasLimit, "200000"),
						sdk.NewAttribute(types.AttributeKeyCallbackSourcePortID, ibctesting.MockPort),
						sdk.NewAttribute(types.AttributeKeyCallbackSourceChannelID, ibctesting.FirstChannelID),
						sdk.NewAttribute(types.AttributeKeyCallbackSequence, "1"),
						sdk.NewAttribute(types.AttributeKeyCallbackResult, types.AttributeValueCallbackFailure),
						sdk.NewAttribute(types.AttributeKeyCallbackError, types.ErrNotPacketDataProvider.Error()),
						sdk.NewAttribute(types.AttributeKeyCallbackBaseApplicationVersion, transfertypes.V1),
					),
				}.ToABCIEvents()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			newCtx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
			switch tc.callbackType {
			case types.CallbackTypeReceivePacket:
				types.EmitCallbackEvent(
					newCtx, tc.packet.GetDestPort(), tc.packet.GetDestChannel(),
					tc.packet.GetSequence(), tc.callbackType, tc.callbackData, tc.callbackError,
				)

			default:
				types.EmitCallbackEvent(
					newCtx, tc.packet.GetSourcePort(), tc.packet.GetSourceChannel(),
					tc.packet.GetSequence(), tc.callbackType, tc.callbackData, tc.callbackError,
				)
			}

			actualEvents := newCtx.EventManager().Events().ToABCIEvents()
			expectedEvents := tc.expectedEvents()

			ibctesting.AssertEvents(&s.Suite, expectedEvents, actualEvents)
		})
	}
}
