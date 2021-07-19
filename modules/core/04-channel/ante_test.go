package channel_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channel "github.com/cosmos/ibc-go/modules/core/04-channel"
	"github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/testing/mock"
)

func (suite *ChannelTestSuite) TestAnteDecorator() {
	testCases := []struct {
		name     string
		malleate func(suite *ChannelTestSuite) []sdk.Msg
		expPass  bool
	}{
		{
			"success on single msg",
			func(suite *ChannelTestSuite) []sdk.Msg {
				packet := types.NewPacket([]byte(mock.MockPacketData), 1,
					suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
					suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
					clienttypes.NewHeight(1, 0), 0)

				return []sdk.Msg{types.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer")}
			},
			true,
		},
		{
			"success on multiple msgs",
			func(suite *ChannelTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				for i := 1; i <= 5; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						clienttypes.NewHeight(1, 0), 0)

					msgs = append(msgs, types.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				return msgs
			},
			true,
		},
		{
			"success on multiple msgs: 1 fresh recv packet",
			func(suite *ChannelTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				for i := 1; i <= 5; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						clienttypes.NewHeight(1, 0), 0)

					err := suite.path.EndpointA.SendPacket(packet)
					suite.Require().NoError(err)

					// receive all sequences except packet 3
					if i != 3 {
						err = suite.path.EndpointB.RecvPacket(packet)
						suite.Require().NoError(err)
					}

					msgs = append(msgs, types.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}

				return msgs
			},
			true,
		},
		{
			"success on multiple mixed msgs",
			func(suite *ChannelTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						clienttypes.NewHeight(1, 0), 0)
					err := suite.path.EndpointA.SendPacket(packet)
					suite.Require().NoError(err)

					msgs = append(msgs, types.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						clienttypes.NewHeight(1, 0), 0)
					err := suite.path.EndpointB.SendPacket(packet)
					suite.Require().NoError(err)

					msgs = append(msgs, types.NewMsgAcknowledgement(packet, []byte("ack"), []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 4; i <= 6; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						clienttypes.NewHeight(1, 0), 0)
					err := suite.path.EndpointB.SendPacket(packet)
					suite.Require().NoError(err)

					msgs = append(msgs, types.NewMsgTimeout(packet, uint64(i), []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				return msgs
			},
			true,
		},
		{
			"success on multiple mixed msgs: 1 fresh packet of each type",
			func(suite *ChannelTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						clienttypes.NewHeight(1, 0), 0)
					err := suite.path.EndpointA.SendPacket(packet)
					suite.Require().NoError(err)

					// receive all sequences except packet 3
					if i != 3 {

						err := suite.path.EndpointB.RecvPacket(packet)
						suite.Require().NoError(err)
					}

					msgs = append(msgs, types.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						clienttypes.NewHeight(1, 0), 0)
					err := suite.path.EndpointB.SendPacket(packet)
					suite.Require().NoError(err)

					// receive all acks except ack 2
					if i != 2 {
						err = suite.path.EndpointA.RecvPacket(packet)
						suite.Require().NoError(err)
						err = suite.path.EndpointB.AcknowledgePacket(packet, mock.MockAcknowledgement.Acknowledgement())
						suite.Require().NoError(err)
					}

					msgs = append(msgs, types.NewMsgAcknowledgement(packet, []byte("ack"), []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 4; i <= 6; i++ {
					height := suite.chainA.LastHeader.GetHeight()
					timeoutHeight := clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+1)
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						timeoutHeight, 0)
					err := suite.path.EndpointB.SendPacket(packet)
					suite.Require().NoError(err)

					// timeout packet
					suite.coordinator.CommitNBlocks(suite.chainA, 3)

					// timeout packets except sequence 5
					if i != 5 {
						suite.path.EndpointB.UpdateClient()
						err = suite.path.EndpointB.TimeoutPacket(packet)
						suite.Require().NoError(err)
					}

					msgs = append(msgs, types.NewMsgTimeout(packet, uint64(i), []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				return msgs
			},
			true,
		},
		{
			"success on multiple mixed msgs: only 1 fresh msg in total",
			func(suite *ChannelTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						clienttypes.NewHeight(1, 0), 0)

					// receive all packets
					suite.path.EndpointA.SendPacket(packet)
					suite.path.EndpointB.RecvPacket(packet)

					msgs = append(msgs, types.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						clienttypes.NewHeight(1, 0), 0)

					// receive all acks
					suite.path.EndpointB.SendPacket(packet)
					suite.path.EndpointA.RecvPacket(packet)
					suite.path.EndpointB.AcknowledgePacket(packet, mock.MockAcknowledgement.Acknowledgement())

					msgs = append(msgs, types.NewMsgAcknowledgement(packet, []byte("ack"), []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 4; i < 5; i++ {
					height := suite.chainA.LastHeader.GetHeight()
					timeoutHeight := clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+1)
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						timeoutHeight, 0)

					// do not timeout packet, timeout msg is fresh
					suite.path.EndpointB.SendPacket(packet)

					msgs = append(msgs, types.NewMsgTimeout(packet, uint64(i), []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				return msgs
			},
			true,
		},
		{
			"success on single update client msg",
			func(suite *ChannelTestSuite) []sdk.Msg {
				return []sdk.Msg{&clienttypes.MsgUpdateClient{}}
			},
			true,
		},
		{
			"success on multiple update clients",
			func(suite *ChannelTestSuite) []sdk.Msg {
				return []sdk.Msg{&clienttypes.MsgUpdateClient{}, &clienttypes.MsgUpdateClient{}, &clienttypes.MsgUpdateClient{}}
			},
			true,
		},
		{
			"success on multiple update clients and fresh packet message",
			func(suite *ChannelTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{&clienttypes.MsgUpdateClient{}, &clienttypes.MsgUpdateClient{}, &clienttypes.MsgUpdateClient{}}

				packet := types.NewPacket([]byte(mock.MockPacketData), 1,
					suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
					suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
					clienttypes.NewHeight(1, 0), 0)

				return append(msgs, types.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
			},
			true,
		},
		{
			"success of tx with different msg type even if all packet messages are redundant",
			func(suite *ChannelTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{&clienttypes.MsgUpdateClient{}}

				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						clienttypes.NewHeight(1, 0), 0)

					// receive all packets
					suite.path.EndpointA.SendPacket(packet)
					suite.path.EndpointB.RecvPacket(packet)

					msgs = append(msgs, types.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						clienttypes.NewHeight(1, 0), 0)

					// receive all acks
					suite.path.EndpointB.SendPacket(packet)
					suite.path.EndpointA.RecvPacket(packet)
					suite.path.EndpointB.AcknowledgePacket(packet, mock.MockAcknowledgement.Acknowledgement())

					msgs = append(msgs, types.NewMsgAcknowledgement(packet, []byte("ack"), []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 4; i < 6; i++ {
					height := suite.chainA.LastHeader.GetHeight()
					timeoutHeight := clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+1)
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						timeoutHeight, 0)

					err := suite.path.EndpointB.SendPacket(packet)
					suite.Require().NoError(err)

					// timeout packet
					suite.coordinator.CommitNBlocks(suite.chainA, 3)

					suite.path.EndpointB.UpdateClient()
					suite.path.EndpointB.TimeoutPacket(packet)

					msgs = append(msgs, types.NewMsgTimeoutOnClose(packet, uint64(i), []byte("proof"), []byte("channelProof"), clienttypes.NewHeight(0, 1), "signer"))
				}

				// append non packet and update message to msgs to ensure multimsg tx should pass
				msgs = append(msgs, &clienttypes.MsgSubmitMisbehaviour{})

				return msgs
			},
			true,
		},
		{
			"no success on multiple mixed message: all are redundant",
			func(suite *ChannelTestSuite) []sdk.Msg {
				var msgs []sdk.Msg

				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						clienttypes.NewHeight(1, 0), 0)

					// receive all packets
					suite.path.EndpointA.SendPacket(packet)
					suite.path.EndpointB.RecvPacket(packet)

					msgs = append(msgs, types.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						clienttypes.NewHeight(1, 0), 0)

					// receive all acks
					suite.path.EndpointB.SendPacket(packet)
					suite.path.EndpointA.RecvPacket(packet)
					suite.path.EndpointB.AcknowledgePacket(packet, mock.MockAcknowledgement.Acknowledgement())

					msgs = append(msgs, types.NewMsgAcknowledgement(packet, []byte("ack"), []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 4; i < 6; i++ {
					height := suite.chainA.LastHeader.GetHeight()
					timeoutHeight := clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+1)
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						timeoutHeight, 0)

					err := suite.path.EndpointB.SendPacket(packet)
					suite.Require().NoError(err)

					// timeout packet
					suite.coordinator.CommitNBlocks(suite.chainA, 3)

					suite.path.EndpointB.UpdateClient()
					suite.path.EndpointB.TimeoutPacket(packet)

					msgs = append(msgs, types.NewMsgTimeoutOnClose(packet, uint64(i), []byte("proof"), []byte("channelProof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				return msgs
			},
			false,
		},
		{
			"no success if msgs contain update clients and redundant packet messages",
			func(suite *ChannelTestSuite) []sdk.Msg {
				msgs := []sdk.Msg{&clienttypes.MsgUpdateClient{}, &clienttypes.MsgUpdateClient{}, &clienttypes.MsgUpdateClient{}}

				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						clienttypes.NewHeight(1, 0), 0)

					// receive all packets
					suite.path.EndpointA.SendPacket(packet)
					suite.path.EndpointB.RecvPacket(packet)

					msgs = append(msgs, types.NewMsgRecvPacket(packet, []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 1; i <= 3; i++ {
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						clienttypes.NewHeight(1, 0), 0)

					// receive all acks
					suite.path.EndpointB.SendPacket(packet)
					suite.path.EndpointA.RecvPacket(packet)
					suite.path.EndpointB.AcknowledgePacket(packet, mock.MockAcknowledgement.Acknowledgement())

					msgs = append(msgs, types.NewMsgAcknowledgement(packet, []byte("ack"), []byte("proof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				for i := 4; i < 6; i++ {
					height := suite.chainA.LastHeader.GetHeight()
					timeoutHeight := clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+1)
					packet := types.NewPacket([]byte(mock.MockPacketData), uint64(i),
						suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID,
						suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
						timeoutHeight, 0)

					err := suite.path.EndpointB.SendPacket(packet)
					suite.Require().NoError(err)

					// timeout packet
					suite.coordinator.CommitNBlocks(suite.chainA, 3)

					suite.path.EndpointB.UpdateClient()
					suite.path.EndpointB.TimeoutPacket(packet)

					msgs = append(msgs, types.NewMsgTimeoutOnClose(packet, uint64(i), []byte("proof"), []byte("channelProof"), clienttypes.NewHeight(0, 1), "signer"))
				}
				return msgs
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupTest()

			k := suite.chainB.App.GetIBCKeeper().ChannelKeeper
			decorator := channel.NewChannelAnteDecorator(k)

			msgs := tc.malleate(suite)

			deliverCtx := suite.chainB.GetContext().WithIsCheckTx(false)
			checkCtx := suite.chainB.GetContext().WithIsCheckTx(true)

			// create multimsg tx
			txBuilder := suite.chainB.TxConfig.NewTxBuilder()
			err := txBuilder.SetMsgs(msgs...)
			suite.Require().NoError(err)
			tx := txBuilder.GetTx()

			next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) { return ctx, nil }

			_, err = decorator.AnteHandle(deliverCtx, tx, false, next)
			suite.Require().NoError(err, "antedecorator should not error on DeliverTx")

			_, err = decorator.AnteHandle(checkCtx, tx, false, next)
			if tc.expPass {
				suite.Require().NoError(err, "non-strict decorator did not pass as expected")
			} else {
				suite.Require().Error(err, "non-strict antehandler did not return error as expected")
			}
		})
	}
}
