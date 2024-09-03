package transfer_test

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

const mockModuleV2A = mock.ModuleNameV2 + "A"
const mockModuleV2B = mock.ModuleNameV2 + "B"

var (
	asyncRecvPacketResult = channeltypes.RecvPacketResult{
		Status:          channeltypes.PacketStatus_Async,
		Acknowledgement: channeltypes.NewResultAcknowledgement([]byte("async")).Acknowledgement(),
	}
	successRecvPacketResult = channeltypes.RecvPacketResult{
		Status:          channeltypes.PacketStatus_Success,
		Acknowledgement: channeltypes.NewResultAcknowledgement([]byte("success")).Acknowledgement(),
	}
	failedRecvPacketResult = channeltypes.RecvPacketResult{
		Status:          channeltypes.PacketStatus_Failure,
		Acknowledgement: channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed ack")).Acknowledgement(),
	}
)

func (suite *TransferTestSuite) TestIBCModuleV2SyncHappyPath() {
	var (
		path             *ibctesting.Path
		data             []channeltypes.PacketData
		expectedMultiAck channeltypes.MultiAcknowledgement
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success", func() {}, nil,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.SetupV2()

			data = []channeltypes.PacketData{
				{
					AppName: types.ModuleName,
					Payload: channeltypes.Payload{
						Version: types.V2,
						Value:   getTransferPacketBz(suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String()),
					},
				},
			}

			expectedMultiAck = channeltypes.MultiAcknowledgement{
				AcknowledgementResults: []channeltypes.AcknowledgementResult{
					{
						AppName: types.ModuleName,
						RecvPacketResult: channeltypes.RecvPacketResult{
							Status:          channeltypes.PacketStatus_Success,
							Acknowledgement: channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement(),
						},
					},
				},
			}

			tc.malleate()

			timeoutHeight := suite.chainA.GetTimeoutHeight()

			sequence, err := path.EndpointA.SendPacketV2POC(timeoutHeight, 0, data)
			suite.Require().NoError(err)

			packet := channeltypes.NewPacketV2(data, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ClientID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, timeoutHeight, 0)

			err = path.EndpointB.RecvPacketV2(packet)
			suite.Require().NoError(err)

			_, ok := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetMultiAcknowledgement(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, packet.GetSequence())

			suite.Require().False(ok, "multi ack should not be written in sync case")
			err = path.EndpointA.AcknowledgePacketV2(packet, expectedMultiAck)
			suite.Require().NoError(err)

		})
	}
}

func (suite *TransferTestSuite) TestIBCModuleV2Async() {
	var (
		path             *ibctesting.Path
		data             []channeltypes.PacketData
		expectedMultiAck channeltypes.MultiAcknowledgement

		// asyncAckModuleAFn and asyncAckModuleBFn are functions which can be defined which will simulate the asynchronous writing
		// of an acknowledgement by the mock applications A & B.
		asyncAckModuleAFn func(channeltypes.PacketV2) error
		asyncAckModuleBFn func(channeltypes.PacketV2) error
	)

	testCases := []struct {
		name        string
		malleate    func()
		expAckError error
	}{
		{
			"success: single async app A", func() {
			// update mock moduleA to return an async result
			expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = asyncRecvPacketResult

			// simulate an async app
			suite.chainB.GetSimApp().MockV2ModuleA.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}

			// at a future point, the application writes the async acknowledgement
			asyncAckModuleAFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = successRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2A, successRecvPacketResult)
			}
		}, nil,
		},
		{
			"success: single async app A writes failed ack", func() {
			// update mock moduleA to return an async result
			expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = asyncRecvPacketResult

			// simulate an async app
			suite.chainB.GetSimApp().MockV2ModuleA.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}

			// at a future point, the application writes the async acknowledgement
			asyncAckModuleAFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = failedRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2A, failedRecvPacketResult)
			}
		}, nil,
		},
		{
			"success: single async app B", func() {
			// update mock moduleA to return an async result
			expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = asyncRecvPacketResult

			// simulate an async app
			suite.chainB.GetSimApp().MockV2ModuleB.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}

			// at a future point, the application writes the async acknowledgement
			asyncAckModuleAFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = successRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2B, successRecvPacketResult)
			}
		}, nil,
		},
		{
			"success: single async app B writes failed ack", func() {
			// update mock moduleA to return an async result
			expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = asyncRecvPacketResult

			// simulate an async app
			suite.chainB.GetSimApp().MockV2ModuleB.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}

			// at a future point, the application writes the async acknowledgement
			asyncAckModuleAFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = failedRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2B, failedRecvPacketResult)
			}
		}, nil,
		},
		{
			"success: two async apps A & B", func() {

			// update mock moduleA to return an async result
			expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = asyncRecvPacketResult
			expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = asyncRecvPacketResult

			// simulate an async app
			suite.chainB.GetSimApp().MockV2ModuleA.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}
			suite.chainB.GetSimApp().MockV2ModuleB.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}

			// at a future point, the application writes the async acknowledgement
			asyncAckModuleAFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = successRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2A, successRecvPacketResult)
			}

			asyncAckModuleBFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = successRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2B, successRecvPacketResult)
			}
		}, nil,
		},
		{
			"success: two async apps A & B both write failed acks", func() {

			// update mock moduleA to return an async result
			expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = asyncRecvPacketResult
			expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = asyncRecvPacketResult

			// simulate an async app
			suite.chainB.GetSimApp().MockV2ModuleA.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}
			suite.chainB.GetSimApp().MockV2ModuleB.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}

			// at a future point, the application writes the async acknowledgement
			asyncAckModuleAFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = failedRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2A, failedRecvPacketResult)
			}
			asyncAckModuleBFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = failedRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2B, failedRecvPacketResult)
			}
		}, nil,
		},
		{
			"success: two async apps A & B one success ack one failed ack", func() {

			// update mock moduleA to return an async result
			expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = asyncRecvPacketResult
			expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = asyncRecvPacketResult

			// simulate an async app
			suite.chainB.GetSimApp().MockV2ModuleA.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}
			suite.chainB.GetSimApp().MockV2ModuleB.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}

			// at a future point, the application writes the async acknowledgement
			asyncAckModuleAFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = successRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2A, successRecvPacketResult)
			}
			asyncAckModuleBFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = failedRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2B, failedRecvPacketResult)
			}
		}, nil,
		},
		{
			"failure: two async apps, only one writes ack", func() {

			// update mock moduleA to return an async result
			expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = asyncRecvPacketResult
			expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = asyncRecvPacketResult

			// simulate an async app
			suite.chainB.GetSimApp().MockV2ModuleA.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}
			suite.chainB.GetSimApp().MockV2ModuleB.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}

			// NOTE: mock module A never writes an ack
			asyncAckModuleAFn = func(channeltypes.PacketV2) error {
				return nil
			}

			// at a future point, the application writes the async acknowledgement
			asyncAckModuleBFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = successRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2B, successRecvPacketResult)
			}
			// an invalid proof error is returned since the ack has not been written on chainB by mock module A yet.
		}, commitmenttypes.ErrInvalidProof,
		},
		{
			"failure: two async apps, only one writes failed ack", func() {

			// update mock moduleA to return an async result
			expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = asyncRecvPacketResult
			expectedMultiAck.AcknowledgementResults[2].RecvPacketResult = asyncRecvPacketResult

			// simulate an async app
			suite.chainB.GetSimApp().MockV2ModuleA.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}
			suite.chainB.GetSimApp().MockV2ModuleB.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
				return asyncRecvPacketResult
			}

			// at a future point, the application writes the async acknowledgement
			asyncAckModuleAFn = func(packet channeltypes.PacketV2) error {
				expectedMultiAck.AcknowledgementResults[1].RecvPacketResult = failedRecvPacketResult
				return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mockModuleV2A, failedRecvPacketResult)
			}

			// NOTE: mock module B never writes an ack
			asyncAckModuleBFn = func(packet channeltypes.PacketV2) error {
				return nil
			}
			// an invalid proof error is returned since the ack has not been written on chainB by mock module A yet.
		}, commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			asyncAckModuleAFn = nil
			asyncAckModuleBFn = nil

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.SetupV2()

			data = []channeltypes.PacketData{
				{
					AppName: types.ModuleName,
					Payload: channeltypes.Payload{
						Version: types.V2,
						Value:   getTransferPacketBz(suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String()),
					},
				},
				{
					AppName: mockModuleV2A,
					Payload: channeltypes.Payload{
						Version: mock.Version,
						Value:   mock.MockPacketData,
					},
				},
				{
					AppName: mockModuleV2B,
					Payload: channeltypes.Payload{
						Version: mock.Version,
						Value:   mock.MockPacketData,
					},
				},
			}

			// By default, it is assumed that all acks will eventually be written and be successful.
			// Each test case can modify this as required.
			expectedMultiAck = channeltypes.MultiAcknowledgement{
				AcknowledgementResults: []channeltypes.AcknowledgementResult{
					{
						AppName: types.ModuleName,
						RecvPacketResult: channeltypes.RecvPacketResult{
							Status:          channeltypes.PacketStatus_Success,
							Acknowledgement: channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement(),
						},
					},
					{
						AppName: mockModuleV2A,
						RecvPacketResult: channeltypes.RecvPacketResult{
							Status:          channeltypes.PacketStatus_Success,
							Acknowledgement: channeltypes.NewResultAcknowledgement([]byte("success")).Acknowledgement(),
						},
					},
					{
						AppName: mockModuleV2B,
						RecvPacketResult: channeltypes.RecvPacketResult{
							Status:          channeltypes.PacketStatus_Success,
							Acknowledgement: channeltypes.NewResultAcknowledgement([]byte("success")).Acknowledgement(),
						},
					},
				},
			}

			tc.malleate()

			timeoutHeight := suite.chainA.GetTimeoutHeight()

			sequence, err := path.EndpointA.SendPacketV2POC(timeoutHeight, 0, data)
			suite.Require().NoError(err)

			packet := channeltypes.NewPacketV2(data, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ClientID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, timeoutHeight, 0)

			err = path.EndpointB.RecvPacketV2(packet)
			suite.Require().NoError(err)

			storedMultiAck, ok := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetMultiAcknowledgement(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, packet.GetSequence())
			suite.Require().True(ok)
			suite.Require().Equal(expectedMultiAck, storedMultiAck, "stored multi ack is not as expected")

			// if either module has specified an async write fn, it will be executed.
			if asyncAckModuleAFn != nil {
				err = asyncAckModuleAFn(packet)
				suite.Require().NoError(err)
			}

			if asyncAckModuleBFn != nil {
				err = asyncAckModuleBFn(packet)
				suite.Require().NoError(err)
			}

			// update clients to ensure any state changes are visible.
			suite.Require().NoError(path.EndpointB.UpdateClient())
			suite.Require().NoError(path.EndpointA.UpdateClient())

			err = path.EndpointA.AcknowledgePacketV2(packet, expectedMultiAck)
			if tc.expAckError != nil {
				suite.Require().Contains(err.Error(), tc.expAckError.Error())
			} else {
				suite.Require().NoError(err)
			}

		})
	}
}

// getTransferPacketBz returns the packet data bytes for the transfer app.
// the actual values aren't so important for the test, so we just use a fixed value.
func getTransferPacketBz(sender, receiver string) []byte {
	ftpd := types.FungibleTokenPacketDataV2{
		Tokens: []types.Token{
			{
				Denom:  types.NewDenom(ibctesting.TestCoin.Denom),
				Amount: "1000",
			},
		},
		Sender:     sender,
		Receiver:   receiver,
		Memo:       "",
		Forwarding: types.ForwardingPacketData{},
	}

	bz, err := ftpd.Marshal()
	if err != nil {
		panic(err)
	}

	return bz
}
