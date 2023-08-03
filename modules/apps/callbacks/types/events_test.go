package types_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

func (s *CallbacksTypesTestSuite) TestLogger() {
	s.SetupSuite()

	mockLogger := ibcmock.NewMockLogger()
	ctx := s.chain.GetContext().WithLogger(mockLogger)
	types.Logger(ctx)

	s.Require().Equal(mockLogger.WithRecord, []interface{}{"module", "x/" + types.ModuleName})
}

func (s *CallbacksTypesTestSuite) TestEvents() {
	testCases := []struct {
		name          string
		packet        channeltypes.Packet
		callbackType  types.CallbackType
		callbackData  types.CallbackData
		callbackError error
		expEvents     ibctesting.EventsMap
	}{
		{
			"success: ack callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeAcknowledgement,
			types.CallbackData{
				ContractAddr:   ibctesting.TestAccAddress,
				GasLimit:       100000,
				CommitGasLimit: 200000,
			},
			nil,
			ibctesting.EventsMap{
				types.EventTypeSourceCallback: {
					sdk.AttributeKeyModule:                    types.ModuleName,
					types.AttributeKeyCallbackTrigger:         string(types.CallbackTypeAcknowledgement),
					types.AttributeKeyCallbackAddress:         ibctesting.TestAccAddress,
					types.AttributeKeyCallbackGasLimit:        "100000",
					types.AttributeKeyCallbackCommitGasLimit:  "200000",
					types.AttributeKeyCallbackSourcePortID:    ibctesting.MockPort,
					types.AttributeKeyCallbackSourceChannelID: ibctesting.FirstChannelID,
					types.AttributeKeyCallbackSequence:        "1",
					types.AttributeKeyCallbackResult:          types.AttributeValueCallbackSuccess,
				},
			},
		},
		{
			"success: send packet callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeSendPacket,
			types.CallbackData{
				ContractAddr:   ibctesting.TestAccAddress,
				GasLimit:       100000,
				CommitGasLimit: 200000,
			},
			nil,
			ibctesting.EventsMap{
				types.EventTypeSourceCallback: {
					sdk.AttributeKeyModule:                    types.ModuleName,
					types.AttributeKeyCallbackTrigger:         string(types.CallbackTypeSendPacket),
					types.AttributeKeyCallbackAddress:         ibctesting.TestAccAddress,
					types.AttributeKeyCallbackGasLimit:        "100000",
					types.AttributeKeyCallbackCommitGasLimit:  "200000",
					types.AttributeKeyCallbackSourcePortID:    ibctesting.MockPort,
					types.AttributeKeyCallbackSourceChannelID: ibctesting.FirstChannelID,
					types.AttributeKeyCallbackSequence:        "1",
					types.AttributeKeyCallbackResult:          types.AttributeValueCallbackSuccess,
				},
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
				ContractAddr:   ibctesting.TestAccAddress,
				GasLimit:       100000,
				CommitGasLimit: 200000,
			},
			nil,
			ibctesting.EventsMap{
				types.EventTypeSourceCallback: {
					sdk.AttributeKeyModule:                    types.ModuleName,
					types.AttributeKeyCallbackTrigger:         string(types.CallbackTypeTimeoutPacket),
					types.AttributeKeyCallbackAddress:         ibctesting.TestAccAddress,
					types.AttributeKeyCallbackGasLimit:        "100000",
					types.AttributeKeyCallbackCommitGasLimit:  "200000",
					types.AttributeKeyCallbackSourcePortID:    ibctesting.MockPort,
					types.AttributeKeyCallbackSourceChannelID: ibctesting.FirstChannelID,
					types.AttributeKeyCallbackSequence:        "1",
					types.AttributeKeyCallbackResult:          types.AttributeValueCallbackSuccess,
				},
			},
		},
		{
			"success: timeout callback",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeWriteAcknowledgement,
			types.CallbackData{
				ContractAddr:   ibctesting.TestAccAddress,
				GasLimit:       100000,
				CommitGasLimit: 200000,
			},
			nil,
			ibctesting.EventsMap{
				types.EventTypeDestinationCallback: {
					sdk.AttributeKeyModule:                   types.ModuleName,
					types.AttributeKeyCallbackTrigger:        string(types.CallbackTypeWriteAcknowledgement),
					types.AttributeKeyCallbackAddress:        ibctesting.TestAccAddress,
					types.AttributeKeyCallbackGasLimit:       "100000",
					types.AttributeKeyCallbackCommitGasLimit: "200000",
					types.AttributeKeyCallbackDestPortID:     ibctesting.MockFeePort,
					types.AttributeKeyCallbackDestChannelID:  ibctesting.InvalidID,
					types.AttributeKeyCallbackSequence:       "1",
					types.AttributeKeyCallbackResult:         types.AttributeValueCallbackSuccess,
				},
			},
		},
		{
			"success: unknown callback, unreachable code",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			"something",
			types.CallbackData{
				ContractAddr:   ibctesting.TestAccAddress,
				GasLimit:       100000,
				CommitGasLimit: 200000,
			},
			nil,
			ibctesting.EventsMap{
				"unknown": {
					sdk.AttributeKeyModule:                    types.ModuleName,
					types.AttributeKeyCallbackTrigger:         "something",
					types.AttributeKeyCallbackAddress:         ibctesting.TestAccAddress,
					types.AttributeKeyCallbackGasLimit:        "100000",
					types.AttributeKeyCallbackCommitGasLimit:  "200000",
					types.AttributeKeyCallbackSourcePortID:    ibctesting.MockPort,
					types.AttributeKeyCallbackSourceChannelID: ibctesting.FirstChannelID,
					types.AttributeKeyCallbackSequence:        "1",
					types.AttributeKeyCallbackResult:          types.AttributeValueCallbackSuccess,
				},
			},
		},
		{
			"failure: ack callback with error",
			channeltypes.NewPacket(
				ibctesting.MockPacketData, 1, ibctesting.MockPort, ibctesting.FirstChannelID,
				ibctesting.MockFeePort, ibctesting.InvalidID, clienttypes.NewHeight(1, 100), 0,
			),
			types.CallbackTypeAcknowledgement,
			types.CallbackData{
				ContractAddr:   ibctesting.TestAccAddress,
				GasLimit:       100000,
				CommitGasLimit: 200000,
			},
			types.ErrNotPacketDataProvider,
			ibctesting.EventsMap{
				types.EventTypeSourceCallback: {
					sdk.AttributeKeyModule:                    types.ModuleName,
					types.AttributeKeyCallbackTrigger:         string(types.CallbackTypeAcknowledgement),
					types.AttributeKeyCallbackAddress:         ibctesting.TestAccAddress,
					types.AttributeKeyCallbackGasLimit:        "100000",
					types.AttributeKeyCallbackCommitGasLimit:  "200000",
					types.AttributeKeyCallbackSourcePortID:    ibctesting.MockPort,
					types.AttributeKeyCallbackSourceChannelID: ibctesting.FirstChannelID,
					types.AttributeKeyCallbackSequence:        "1",
					types.AttributeKeyCallbackResult:          types.AttributeValueCallbackFailure,
					types.AttributeKeyCallbackError:           types.ErrNotPacketDataProvider.Error(),
				},
			},
		},
	}

	for _, tc := range testCases {
		newCtx := sdk.Context{}.WithEventManager(sdk.NewEventManager())

		types.EmitCallbackEvent(newCtx, tc.packet, tc.callbackType, tc.callbackData, tc.callbackError)
		events := newCtx.EventManager().Events().ToABCIEvents()
		ibctesting.AssertEvents(&s.Suite, tc.expEvents, events)
	}
}
