package keeper_test

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	mock "github.com/cosmos/ibc-go/v9/testing/mock"
	mockv2 "github.com/cosmos/ibc-go/v9/testing/mock/v2"
)

func (suite *KeeperTestSuite) TestSendPacket() {
	var path *ibctesting.Path
	var msg *channeltypesv2.MsgSendPacket

	testCases := []struct {
		name        string
		malleate    func()
		expError    error
		shouldPanic bool
	}{
		{
			name:     "success",
			malleate: func() {},
			expError: nil,
		},
		{
			name: "failure: application callback error",
			malleate: func() {
				path.EndpointA.Chain.GetSimApp().MockModuleV2A.IBCApp.OnSendPacket = func(ctx context.Context, sourceID string, destinationID string, sequence uint64, data channeltypesv2.PacketData, signer sdk.AccAddress) error {
					return mock.MockApplicationCallbackError
				}
			},
			expError: mock.MockApplicationCallbackError,
		},
		{
			name: "failure: invalid client ID",
			malleate: func() {
				msg.SourceId = "foo"
			},
			shouldPanic: true,
		},
		{
			name: "failure: route to non existing app",
			malleate: func() {
				msg.PacketData[0].SourcePort = "foo"
			},
			expError: fmt.Errorf("no route for foo"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			msg = channeltypesv2.NewMsgSendPacket(path.EndpointA.ClientID, suite.chainA.GetTimeoutTimestamp(), suite.chainA.SenderAccount.GetAddress().String(), mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			tc.malleate()

			if tc.shouldPanic {
				suite.Require().Panics(func() { _, _ = path.EndpointA.Chain.SendMsgs(msg) })
				return
			}

			res, err := path.EndpointA.Chain.SendMsgs(msg)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}
