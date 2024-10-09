package keeper_test

import (
	"context"
	"fmt"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
	mockv2 "github.com/cosmos/ibc-go/v9/testing/mock/v2"
)

func (suite *KeeperTestSuite) TestMsgSendPacket() {
	var path *ibctesting.Path
	var msg *channeltypesv2.MsgSendPacket
	var expectedPacket channeltypesv2.Packet

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
			name: "failure: timeout elapsed",
			malleate: func() {
				// ensure a message timeout.
				msg.TimeoutTimestamp = uint64(1)
			},
			expError: channeltypesv1.ErrTimeoutElapsed,
		},
		{
			name: "failure: inactive client",
			malleate: func() {
				path.EndpointA.FreezeClient()
			},
			expError: clienttypes.ErrClientNotActive,
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
			name: "failure: counterparty not found",
			malleate: func() {
				msg.SourceId = "foo"
			},
			expError: fmt.Errorf("counterparty not found"),
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

			timeoutTimestamp := suite.chainA.GetTimeoutTimestamp()
			msg = channeltypesv2.NewMsgSendPacket(path.EndpointA.ClientID, timeoutTimestamp, suite.chainA.SenderAccount.GetAddress().String(), mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			expectedPacket = channeltypesv2.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID, timeoutTimestamp, mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			tc.malleate()

			res, err := path.EndpointA.Chain.SendMsgs(msg)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				ck := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2

				packetCommitment, ok := ck.GetPacketCommitment(path.EndpointA.Chain.GetContext(), path.EndpointA.ClientID, 1)
				suite.Require().True(ok)
				suite.Require().Equal(channeltypesv2.CommitPacket(expectedPacket), []byte(packetCommitment), "packet commitment is not stored correctly")

				nextSequenceSend, ok := ck.GetNextSequenceSend(path.EndpointA.Chain.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(ok)
				suite.Require().Equal(uint64(2), nextSequenceSend, "next sequence send was not incremented correctly")

			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgRecvPacket() {
	var path *ibctesting.Path
	var msg *channeltypesv2.MsgRecvPacket
	var recvPacket channeltypesv2.Packet

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
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			timeoutTimestamp := suite.chainA.GetTimeoutTimestamp()
			msgSendPacket := channeltypesv2.NewMsgSendPacket(path.EndpointA.ClientID, timeoutTimestamp, suite.chainA.SenderAccount.GetAddress().String(), mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			res, err := path.EndpointA.Chain.SendMsgs(msgSendPacket)
			suite.Require().NoError(err)
			suite.Require().NotNil(res)

			suite.Require().NoError(path.EndpointB.UpdateClient())

			recvPacket = channeltypesv2.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID, timeoutTimestamp, mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			tc.malleate()

			// get proof of packet commitment from chainA
			packetKey := hostv2.PacketCommitmentKey(recvPacket.SourceId, sdk.Uint64ToBigEndian(recvPacket.Sequence))
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			msg = channeltypesv2.NewMsgRecvPacket(recvPacket, proof, proofHeight, suite.chainB.SenderAccount.GetAddress().String())

			res, err = path.EndpointB.Chain.SendMsgs(msg)
			suite.Require().NoError(path.EndpointA.UpdateClient())

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
