package keeper_test

import (
	"errors"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	internalerrors "github.com/cosmos/ibc-go/v10/modules/core/internal/errors"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

var (
	timeoutHeight = clienttypes.NewHeight(1, 10000)
	maxSequence   = uint64(10)
)

// TestRegisterCounterparty tests that counterpartyInfo is correctly stored
// and only if the submittor is the same submittor as prior createClient msg
func (suite *KeeperTestSuite) TestRegisterCounterparty() {
	var path *ibctesting.Path
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				path.SetupClients()
			},
			nil,
		},
		{
			"client not created first",
			func() {},
			ibcerrors.ErrUnauthorized,
		},
		{
			"creator is different than expected",
			func() {
				path.SetupClients()
				path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.SetClientCreator(suite.chainA.GetContext(), path.EndpointA.ClientID, sdk.AccAddress(ibctesting.TestAccAddress))
			},
			ibcerrors.ErrUnauthorized,
		},
	}
	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()
			merklePrefix := [][]byte{[]byte("ibc"), []byte("channel-7")}
			msg := clientv2types.NewMsgRegisterCounterparty(path.EndpointA.ClientID, merklePrefix, path.EndpointB.ClientID, suite.chainA.SenderAccount.GetAddress().String())
			_, err := suite.chainA.App.GetIBCKeeper().RegisterCounterparty(suite.chainA.GetContext(), msg)
			if tc.expError != nil {
				suite.Require().Error(err)
				suite.Require().True(errors.Is(err, tc.expError))
			} else {
				suite.Require().NoError(err)
				counterpartyInfo, ok := suite.chainA.App.GetIBCKeeper().ClientV2Keeper.GetClientCounterparty(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(ok)
				suite.Require().Equal(counterpartyInfo, clientv2types.NewCounterpartyInfo(merklePrefix, path.EndpointB.ClientID))
				nextSeqSend, ok := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetNextSequenceSend(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(ok)
				suite.Require().Equal(nextSeqSend, uint64(1))
			}
		})
	}
}

// tests the IBC handler receiving a packet on ordered and unordered channels.
// It verifies that the storing of an acknowledgement on success occurs. It
// tests high level properties like ordering and basic sanity checks. More
// rigorous testing of 'RecvPacket' can be found in the
// 04-channel/keeper/packet_test.go.
func (suite *KeeperTestSuite) TestHandleRecvPacket() {
	var (
		packet channeltypes.Packet
		path   *ibctesting.Path
	)

	testCases := []struct {
		name      string
		malleate  func()
		expError  error
		expRevert bool
		async     bool // indicate no ack written
		replay    bool // indicate replay (no-op)
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, false, false},
		{"success: UNORDERED", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, false, false},
		{"success: UNORDERED out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, nil, false, false, false},
		{"success: OnRecvPacket callback returns revert=true", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockFailPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockFailPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, true, false, false},
		{"success: ORDERED - async acknowledgement", func() {
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, true, false},
		{"success: UNORDERED - async acknowledgement", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibcmock.MockAsyncPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibcmock.MockAsyncPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, nil, false, true, false},
		{"failure: ORDERED out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to receive packet with sequence 10 without receiving packet with sequence 1
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}
		}, errors.New("packet sequence is out of order"), false, false, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)
		}, errors.New("channel not found"), false, false, false},
		{"packet not sent", func() {
			path.Setup()
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, errors.New("receive packet verification failed: couldn't verify counterparty packet commitment"), false, false, false},
		{"successful no-op: ORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, nil, false, false, true},
		{"successful no-op: UNORDERED - packet already received (replay)", func() {
			// mock will panic if application callback is called twice on the same packet
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, nil, false, false, true},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)

			// get proof of packet commitment from chainA
			packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			if path.EndpointA.ChannelID != "" {
				proof, proofHeight = path.EndpointA.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, suite.chainB.SenderAccount.GetAddress().String())

			ctx := suite.chainB.GetContext()
			_, err := suite.chainB.App.GetIBCKeeper().RecvPacket(ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expError == nil {
				suite.Require().NoError(err)

				// replay should not fail since it will be treated as a no-op
				_, err := suite.chainB.App.GetIBCKeeper().RecvPacket(suite.chainB.GetContext(), msg)
				suite.Require().NoError(err)

				if tc.expRevert {
					// context events should contain error events
					suite.Require().Contains(events, internalerrors.ConvertToErrorEvents(sdk.Events{ibcmock.NewMockRecvPacketEvent()})[0])
					suite.Require().NotContains(events, ibcmock.NewMockRecvPacketEvent())
				} else {
					if tc.replay {
						// context should not contain application events
						suite.Require().NotContains(events, ibcmock.NewMockRecvPacketEvent())
						suite.Require().NotContains(events, internalerrors.ConvertToErrorEvents(sdk.Events{ibcmock.NewMockRecvPacketEvent()})[0])
					} else {
						// context events should contain application events
						suite.Require().Contains(events, ibcmock.NewMockRecvPacketEvent())
					}
				}

				// verify if ack was written
				ack, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

				if tc.async {
					suite.Require().Nil(ack)
					suite.Require().False(found)

				} else {
					suite.Require().NotNil(ack)
					suite.Require().True(found)
				}
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpdateClient() {
	var path *ibctesting.Path
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: update client, no params",
			func() {},
			nil,
		},
		{
			"success: update client, with v2 params set to correct relayer",
			func() {
				creator := suite.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientV2Params(path.EndpointA.ClientID, creator.String(), clientv2types.NewParams(suite.chainB.SenderAccount.GetAddress().String(), creator.String()))
				_, err := suite.chainA.App.GetIBCKeeper().UpdateClientV2Params(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"failure: update client with invalid relayer",
			func() {
				creator := suite.chainA.SenderAccount.GetAddress()
				msg := clientv2types.NewMsgUpdateClientV2Params(path.EndpointA.ClientID, creator.String(), clientv2types.NewParams(suite.chainB.SenderAccount.GetAddress().String()))
				_, err := suite.chainA.App.GetIBCKeeper().UpdateClientV2Params(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			tc.malleate()

			err := path.EndpointA.UpdateClient()

			if tc.expError == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestRecoverClient() {
	var msg *clienttypes.MsgRecoverClient

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: recover client",
			func() {},
			nil,
		},
		{
			"signer doesn't match authority",
			func() {
				msg.Signer = ibctesting.InvalidID
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"invalid subject client",
			func() {
				msg.SubjectClientId = ibctesting.InvalidID
			},
			clienttypes.ErrRouteNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			subjectPath.SetupClients()
			subject := subjectPath.EndpointA.ClientID
			subjectClientState := suite.chainA.GetClientState(subject)

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			substitutePath.SetupClients()
			substitute := substitutePath.EndpointA.ClientID

			// update substitute twice
			err := substitutePath.EndpointA.UpdateClient()
			suite.Require().NoError(err)
			err = substitutePath.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			tmClientState, ok := subjectClientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), subject, tmClientState)

			msg = clienttypes.NewMsgRecoverClient(suite.chainA.App.GetIBCKeeper().GetAuthority(), subject, substitute)

			tc.malleate()

			_, err = suite.chainA.App.GetIBCKeeper().RecoverClient(suite.chainA.GetContext(), msg)

			if tc.expErr == nil {
				suite.Require().NoError(err)

				// Assert that client status is now Active

				lightClientModule, err := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID)
				suite.Require().NoError(err)
				suite.Require().Equal(lightClientModule.Status(suite.chainA.GetContext(), subjectPath.EndpointA.ClientID), exported.Active)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// tests the IBC handler acknowledgement of a packet on ordered and unordered
// channels. It verifies that the deletion of packet commitments from state
// occurs. It test high level properties like ordering and basic sanity
// checks. More rigorous testing of 'AcknowledgePacket'
// can be found in the 04-channel/keeper/packet_test.go.
func (suite *KeeperTestSuite) TestHandleAcknowledgePacket() {
	var (
		packet channeltypes.Packet
		path   *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
		replay   bool // indicate replay (no-op)
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, nil, false},
		{"success: UNORDERED", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, nil, false},
		{"success: UNORDERED acknowledge out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment)
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)
			}
		}, nil, false},
		{"failure: ORDERED acknowledge out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to acknowledge ack with sequence 10 without acknowledging ack with sequence 1 (removing packet commitment
			for i := uint64(1); i < 10; i++ {
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)
			}
		}, errors.New("packet sequence is out of order"), false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)
		}, errors.New("channel not found"), false},
		{"packet not received", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
		}, errors.New("invalid proof"), false},
		{"successful no-op: ORDERED - packet already acknowledged (replay)", func() {
			path.SetChannelOrdered()
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			suite.Require().NoError(err)
		}, nil, true},
		{"successful no-op: UNORDERED - packet already acknowledged (replay)", func() {
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
			suite.Require().NoError(err)
		}, nil, true},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)
			packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			if path.EndpointB.ChannelID != "" {
				proof, proofHeight = path.EndpointB.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgAcknowledgement(packet, ibcmock.MockAcknowledgement.Acknowledgement(), proof, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

			ctx := suite.chainA.GetContext()
			_, err := suite.chainA.App.GetIBCKeeper().Acknowledgement(ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expError == nil {
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

				// replay should not error as it is treated as a no-op
				_, err := suite.chainA.App.GetIBCKeeper().Acknowledgement(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				if tc.replay {
					// context should not contain application events
					suite.Require().NotContains(events, ibcmock.NewMockAckPacketEvent())
				} else {
					// context events should contain application events
					suite.Require().Contains(events, ibcmock.NewMockAckPacketEvent())
				}
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

// tests the IBC handler timing out a packet on ordered and unordered channels.
// It verifies that the deletion of a packet commitment occurs. It tests
// high level properties like ordering and basic sanity checks. More
// rigorous testing of 'TimeoutPacket' and 'TimeoutExecuted' can be found in
// the 04-channel/keeper/timeout_test.go.
func (suite *KeeperTestSuite) TestHandleTimeoutPacket() {
	var (
		packet    channeltypes.Packet
		packetKey []byte
		path      *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
		noop     bool // indicate no-op
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, nil, false},
		{"success: UNORDERED", func() {
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, nil, false},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, nil, false},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, nil, false},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, errors.New("channel not found"), false},
		{"successful no-op: UNORDERED - packet not sent", func() {
			path.Setup()
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		}, nil, true},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			var (
				proof       []byte
				proofHeight clienttypes.Height
			)
			if path.EndpointB.ChannelID != "" {
				proof, proofHeight = path.EndpointB.QueryProof(packetKey)
			}

			msg := channeltypes.NewMsgTimeout(packet, 1, proof, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

			ctx := suite.chainA.GetContext()
			_, err := suite.chainA.App.GetIBCKeeper().Timeout(ctx, msg)

			events := ctx.EventManager().Events()

			if tc.expErr == nil {
				suite.Require().NoError(err)

				// replay should not return an error as it is treated as a no-op
				_, err := suite.chainA.App.GetIBCKeeper().Timeout(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

				if tc.noop {
					// context should not contain application events
					suite.Require().NotContains(events, ibcmock.NewMockTimeoutPacketEvent())
				} else {
					// context should contain application events
					suite.Require().Contains(events, ibcmock.NewMockTimeoutPacketEvent())
				}

			} else {
				suite.Require().Error(err)

				suite.Require().Contains(err.Error(), tc.expErr.Error())
			}
		})
	}
}

// tests the IBC handler timing out a packet via channel closure on ordered
// and unordered channels. It verifies that the deletion of a packet
// commitment occurs. It tests high level properties like ordering and basic
// sanity checks. More rigorous testing of 'TimeoutOnClose' and
// 'TimeoutExecuted' can be found in the 04-channel/keeper/timeout_test.go.
func (suite *KeeperTestSuite) TestHandleTimeoutOnClosePacket() {
	var (
		packet    channeltypes.Packet
		packetKey []byte
		path      *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{"success: ORDERED", func() {
			path.SetChannelOrdered()
			path.Setup()

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"success: UNORDERED", func() {
			path.Setup()

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"success: UNORDERED timeout out of order packet", func() {
			// setup uses an UNORDERED channel
			path.Setup()

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"success: ORDERED timeout out of order packet", func() {
			path.SetChannelOrdered()
			path.Setup()

			// attempts to timeout the last packet sent without timing out the first packet
			// packet sequences begin at 1
			for i := uint64(1); i < maxSequence; i++ {
				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			}

			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"channel does not exist", func() {
			// any non-nil value of packet is valid
			suite.Require().NotNil(packet)

			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, errors.New("channel not found")},
		{"successful no-op: UNORDERED - packet not sent", func() {
			path.Setup()
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 1), 0)
			packetKey = host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			// close counterparty channel
			path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		}, nil},
		{"ORDERED: channel not closed", func() {
			path.SetChannelOrdered()
			path.Setup()

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA client to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)
			packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
		}, errors.New("invalid proof")},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			proof, proofHeight := suite.chainB.QueryProof(packetKey)

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			closedProof, _ := suite.chainB.QueryProof(channelKey)

			msg := channeltypes.NewMsgTimeoutOnClose(packet, 1, proof, closedProof, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

			_, err := suite.chainA.App.GetIBCKeeper().TimeoutOnClose(suite.chainA.GetContext(), msg)

			if tc.expError == nil {
				suite.Require().NoError(err)

				// replay should not return an error as it will be treated as a no-op
				_, err := suite.chainA.App.GetIBCKeeper().TimeoutOnClose(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				// verify packet commitment was deleted on source chain
				has := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasPacketCommitment(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				suite.Require().False(has)

			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpgradeClient() {
	var (
		path              *ibctesting.Path
		newChainID        string
		newClientHeight   clienttypes.Height
		upgradedClient    *ibctm.ClientState
		upgradedConsState exported.ConsensusState
		lastHeight        exported.Height
		msg               *clienttypes.MsgUpgradeClient
	)
	cases := []struct {
		name   string
		setup  func()
		expErr error
	}{
		{
			name: "successful upgrade",
			setup: func() {
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod+ibctesting.TrustingPeriod, ibctesting.MaxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				// Call ZeroCustomFields on upgraded clients to clear any client-chosen parameters in test-case upgradedClient
				upgradedClient = upgradedClient.ZeroCustomFields()

				upgradedConsState = &ibctm.ConsensusState{
					NextValidatorsHash: []byte("nextValsHash"),
				}

				// last Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				upgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)
				upgradedConsStateBz, err := clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), upgradedConsState)
				suite.Require().NoError(err)

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				latestHeight := path.EndpointA.GetClientLatestHeight()
				upgradeClientProof, _ := suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(lastHeight.GetRevisionHeight())), latestHeight.GetRevisionHeight())
				upgradedConsensusStateProof, _ := suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(lastHeight.GetRevisionHeight())), latestHeight.GetRevisionHeight())

				msg, err = clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState,
					upgradeClientProof, upgradedConsensusStateProof, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			expErr: nil,
		},
		{
			name: "VerifyUpgrade fails",
			setup: func() {
				upgradedClient = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod+ibctesting.TrustingPeriod, ibctesting.MaxClockDrift, newClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				// Call ZeroCustomFields on upgraded clients to clear any client-chosen parameters in test-case upgradedClient
				upgradedClient = upgradedClient.ZeroCustomFields()

				upgradedConsState = &ibctm.ConsensusState{
					NextValidatorsHash: []byte("nextValsHash"),
				}

				// last Height is at next block
				lastHeight = clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				upgradedClientBz, err := clienttypes.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClient)
				suite.Require().NoError(err)
				upgradedConsStateBz, err := clienttypes.MarshalConsensusState(suite.chainA.App.AppCodec(), upgradedConsState)
				suite.Require().NoError(err)

				// zero custom fields and store in upgrade store
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedClientBz)            //nolint:errcheck // ignore error for testing
				suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(lastHeight.GetRevisionHeight()), upgradedConsStateBz) //nolint:errcheck // ignore error for testing

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				msg, err = clienttypes.NewMsgUpgradeClient(path.EndpointA.ClientID, upgradedClient, upgradedConsState, nil, nil, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			expErr: errors.New("invalid merkle proof"),
		},
	}

	for _, tc := range cases {
		tc := tc
		path = ibctesting.NewPath(suite.chainA, suite.chainB)
		path.SetupClients()

		var err error
		clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
		suite.Require().True(ok)
		revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

		newChainID, err = clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
		suite.Require().NoError(err)

		newClientHeight = clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1)

		tc.setup()

		ctx := suite.chainA.GetContext()
		_, err = suite.chainA.GetSimApp().GetIBCKeeper().UpgradeClient(ctx, msg)

		if tc.expErr == nil {
			suite.Require().NoError(err, "upgrade handler failed on valid case: %s", tc.name)
			newClient, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
			suite.Require().True(ok)
			newChainSpecifiedClient := newClient.(*ibctm.ClientState).ZeroCustomFields()
			suite.Require().Equal(upgradedClient, newChainSpecifiedClient)

			expectedEvents := sdk.Events{
				sdk.NewEvent(
					clienttypes.EventTypeUpgradeClient,
					sdk.NewAttribute(clienttypes.AttributeKeyClientID, ibctesting.FirstClientID),
					sdk.NewAttribute(clienttypes.AttributeKeyClientType, path.EndpointA.GetClientState().ClientType()),
					sdk.NewAttribute(clienttypes.AttributeKeyConsensusHeight, path.EndpointA.GetClientLatestHeight().String()),
				),
			}.ToABCIEvents()

			expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
			ibctesting.AssertEvents(&suite.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())
		} else {
			suite.Require().Error(err, "upgrade handler passed on invalid case: %s", tc.name)
			suite.Require().Contains(err.Error(), tc.expErr.Error())
		}
	}
}

// TestIBCSoftwareUpgrade tests the IBCSoftwareUpgrade rpc handler
func (suite *KeeperTestSuite) TestIBCSoftwareUpgrade() {
	var msg *clienttypes.MsgIBCSoftwareUpgrade
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: valid authority and client upgrade",
			func() {},
			nil,
		},
		{
			"failure: invalid authority address",
			func() {
				msg.Signer = suite.chainA.SenderAccount.GetAddress().String()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: invalid clientState",
			func() {
				msg.UpgradedClientState = nil
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"failure: failed to schedule client upgrade",
			func() {
				msg.Plan.Height = 0
			},
			sdkerrors.ErrInvalidRequest,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()
			validAuthority := suite.chainA.App.GetIBCKeeper().GetAuthority()
			plan := upgradetypes.Plan{
				Name:   "upgrade IBC clients",
				Height: 1000,
			}
			// update trusting period
			clientState, ok := path.EndpointB.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientState.TrustingPeriod += 100

			var err error
			msg, err = clienttypes.NewMsgIBCSoftwareUpgrade(
				validAuthority,
				plan,
				clientState,
			)

			suite.Require().NoError(err)

			tc.malleate()

			_, err = suite.chainA.App.GetIBCKeeper().IBCSoftwareUpgrade(suite.chainA.GetContext(), msg)

			if tc.expError == nil {
				suite.Require().NoError(err)
				// upgrade plan is stored
				storedPlan, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(suite.chainA.GetContext())
				suite.Require().NoError(err)
				suite.Require().Equal(plan, storedPlan)

				// upgraded client state is stored
				bz, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(suite.chainA.GetContext(), plan.Height)
				suite.Require().NoError(err)
				upgradedClientState, err := clienttypes.UnmarshalClientState(suite.chainA.App.AppCodec(), bz)
				suite.Require().NoError(err)
				suite.Require().Equal(clientState.ZeroCustomFields(), upgradedClientState)
			} else {
				suite.Require().True(errors.Is(err, tc.expError))
			}
		})
	}
}

// TestUpdateClientParams tests the UpdateClientParams rpc handler
func (suite *KeeperTestSuite) TestUpdateClientParams() {
	signer := suite.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name     string
		msg      *clienttypes.MsgUpdateParams
		expError error
	}{
		{
			"success: valid signer and default params",
			clienttypes.NewMsgUpdateParams(signer, clienttypes.DefaultParams()),
			nil,
		},
		{
			"failure: malformed signer address",
			clienttypes.NewMsgUpdateParams(ibctesting.InvalidID, clienttypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: empty signer address",
			clienttypes.NewMsgUpdateParams("", clienttypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: whitespace signer address",
			clienttypes.NewMsgUpdateParams("    ", clienttypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: unauthorized signer address",
			clienttypes.NewMsgUpdateParams(ibctesting.TestAccAddress, clienttypes.DefaultParams()),
			errors.New("unauthorized"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := suite.chainA.App.GetIBCKeeper().UpdateClientParams(suite.chainA.GetContext(), tc.msg)
			if tc.expError == nil {
				suite.Require().NoError(err)
				p := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.msg.Params, p)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

// TestUpdateConnectionParams tests the UpdateConnectionParams rpc handler
func (suite *KeeperTestSuite) TestUpdateConnectionParams() {
	signer := suite.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name   string
		msg    *connectiontypes.MsgUpdateParams
		expErr error
	}{
		{
			"success: valid signer and default params",
			connectiontypes.NewMsgUpdateParams(signer, connectiontypes.DefaultParams()),
			nil,
		},
		{
			"failure: malformed signer address",
			connectiontypes.NewMsgUpdateParams(ibctesting.InvalidID, connectiontypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: empty signer address",
			connectiontypes.NewMsgUpdateParams("", connectiontypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: whitespace signer address",
			connectiontypes.NewMsgUpdateParams("    ", connectiontypes.DefaultParams()),
			errors.New("unauthorized"),
		},
		{
			"failure: unauthorized signer address",
			connectiontypes.NewMsgUpdateParams(ibctesting.TestAccAddress, connectiontypes.DefaultParams()),
			errors.New("unauthorized"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := suite.chainA.App.GetIBCKeeper().UpdateConnectionParams(suite.chainA.GetContext(), tc.msg)
			if tc.expErr == nil {
				suite.Require().NoError(err)
				p := suite.chainA.App.GetIBCKeeper().ConnectionKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.msg.Params, p)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpdateClientV2Params() {
	var (
		path   *ibctesting.Path
		signer string
		params types.Params
	)
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: valid authority and default params",
			func() {
				signer = suite.chainA.App.GetIBCKeeper().GetAuthority()
			},
			nil,
		},
		{
			"success: valid creator and default params",
			func() {
				signer = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientCreator(suite.chainA.GetContext(), path.EndpointA.ClientID).String()
			},
			nil,
		},
		{
			"success: valid authority and custom params",
			func() {
				signer = suite.chainA.App.GetIBCKeeper().GetAuthority()
				params = types.NewParams(suite.chainB.SenderAccount.String(), suite.chainA.SenderAccount.String())
			},
			nil,
		},
		{
			"success: valid creator and default params",
			func() {
				signer = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientCreator(suite.chainA.GetContext(), path.EndpointA.ClientID).String()
				params = types.NewParams(suite.chainB.SenderAccount.String(), suite.chainA.SenderAccount.String())
			},
			nil,
		},
		{
			"success: valid creator and setting params to empty after it has been set",
			func() {
				signer = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientCreator(suite.chainA.GetContext(), path.EndpointA.ClientID).String()
				params = types.NewParams(suite.chainB.SenderAccount.String(), suite.chainA.SenderAccount.String())
				_, err := suite.chainA.App.GetIBCKeeper().UpdateClientV2Params(suite.chainA.GetContext(), types.NewMsgUpdateClientV2Params(path.EndpointA.ClientID, signer, params))
				suite.Require().NoError(err)
				params = types.DefaultParams()
			},
			nil,
		},
		{
			"success: valid creator and setting params to different params after it has been set",
			func() {
				signer = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientCreator(suite.chainA.GetContext(), path.EndpointA.ClientID).String()
				params = types.NewParams(suite.chainA.SenderAccount.String())
				_, err := suite.chainA.App.GetIBCKeeper().UpdateClientV2Params(suite.chainA.GetContext(), types.NewMsgUpdateClientV2Params(path.EndpointA.ClientID, signer, params))
				suite.Require().NoError(err)
				params = types.NewParams(suite.chainB.SenderAccount.String(), suite.chainA.SenderAccount.String())
			},
			nil,
		},
		{
			"failure: invalid signer",
			func() {
				signer = suite.chainB.SenderAccount.GetAddress().String()
				params = types.NewParams(suite.chainB.SenderAccount.String())
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			params = types.DefaultParams()

			tc.malleate()

			msg := types.NewMsgUpdateClientV2Params(path.EndpointA.ClientID, signer, params)
			_, err := suite.chainA.App.GetIBCKeeper().UpdateClientV2Params(suite.chainA.GetContext(), msg)
			if tc.expError == nil {
				suite.Require().NoError(err)
				p := suite.chainA.App.GetIBCKeeper().ClientV2Keeper.GetParams(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().Equal(params, p)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}

		})
	}
}
