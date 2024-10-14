package keeper_test

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
	mockv2 "github.com/cosmos/ibc-go/v9/testing/mock/v2"
)

func (suite *KeeperTestSuite) TestMsgSendPacket() {
	var (
		path           *ibctesting.Path
		msg            *channeltypesv2.MsgSendPacket
		expectedPacket channeltypesv2.Packet
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
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
				msg.SourceChannel = "foo"
			},
			expError: channeltypesv1.ErrChannelNotFound,
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
			msg = channeltypesv2.NewMsgSendPacket(path.EndpointA.ChannelID, timeoutTimestamp, suite.chainA.SenderAccount.GetAddress().String(), mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			expectedPacket = channeltypesv2.NewPacket(1, path.EndpointA.ChannelID, path.EndpointB.ChannelID, timeoutTimestamp, mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB))

			tc.malleate()

			res, err := path.EndpointA.Chain.SendMsgs(msg)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				ck := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeperV2

				packetCommitment := ck.GetPacketCommitment(path.EndpointA.Chain.GetContext(), path.EndpointA.ChannelID, 1)
				suite.Require().NotNil(packetCommitment)
				suite.Require().Equal(channeltypesv2.CommitPacket(expectedPacket), packetCommitment, "packet commitment is not stored correctly")

				nextSequenceSend, ok := ck.GetNextSequenceSend(path.EndpointA.Chain.GetContext(), path.EndpointA.ChannelID)
				suite.Require().True(ok)
				suite.Require().Equal(uint64(2), nextSequenceSend, "next sequence send was not incremented correctly")

			} else {
				suite.Require().Error(err)
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestProvideCounterparty() {
	var (
		path *ibctesting.Path
		msg  *channeltypesv2.MsgProvideCounterparty
	)
	cases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				// set it before handler
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetChannel(suite.chainA.GetContext(), msg.ChannelId, channeltypesv2.NewChannel(path.EndpointA.ClientID, "", ibctesting.MerklePath))
			},
			nil,
		},

		{
			"failure: signer does not match creator",
			func() {
				msg.Signer = path.EndpointB.Chain.SenderAccount.GetAddress().String()
			},
			ibcerrors.ErrUnauthorized,
		},
		/* // Account sequence mismatch, expected 5, got 6. :thinking:
		{
			"failure: counterparty does not already exists",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.ChannelStore(suite.chainA.GetContext(), path.EndpointA.ChannelID).Delete([]byte(channeltypesv2.ChannelKey))
			},
			channeltypesv2.ErrInvalidChannel,
		},
		*/
	}

	for _, tc := range cases {
		tc := tc
		path = ibctesting.NewPath(suite.chainA, suite.chainB)
		path.SetupClients()

		suite.Require().NoError(path.EndpointA.CreateChannel())
		suite.Require().NoError(path.EndpointB.CreateChannel())

		signer := path.EndpointA.Chain.SenderAccount.GetAddress().String()
		msg = channeltypesv2.NewMsgProvideCounterparty(path.EndpointA.ChannelID, path.EndpointB.ChannelID, signer)

		tc.malleate()

		res, err := path.EndpointA.Chain.SendMsgs(msg)

		expPass := tc.expError == nil
		if expPass {
			suite.Require().NotNil(res)
			suite.Require().Nil(err)

			// Assert counterparty channel id filled in and creator deleted
			channel, found := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelID)
			suite.Require().True(found)
			suite.Require().Equal(channel.CounterpartyChannelId, path.EndpointB.ChannelID)

			_, found = suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetCreator(suite.chainA.GetContext(), path.EndpointA.ChannelID)
			suite.Require().False(found)

			seq, found := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetNextSequenceSend(suite.chainA.GetContext(), path.EndpointA.ChannelID)
			suite.Require().True(found)
			suite.Require().Equal(seq, uint64(1))
		} else {
			suite.Require().Error(err)
		}
	}
}
