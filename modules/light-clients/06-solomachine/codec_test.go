package solomachine_test

import (
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v3/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	solomachine "github.com/cosmos/ibc-go/v3/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite SoloMachineTestSuite) TestUnmarshalDataByType() {
	var (
		data []byte
		err  error
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		cdc := suite.chainA.App.AppCodec()
		cases := []struct {
			name     string
			dataType solomachine.DataType
			malleate func()
			expPass  bool
		}{
			{
				"empty data", solomachine.CLIENT, func() {
					data = []byte{}
				}, false,
			},
			{
				"unspecified", solomachine.UNSPECIFIED, func() {
					path := sm.GetClientStatePath(counterpartyClientIdentifier)
					data, err = solomachine.ClientStateDataBytes(cdc, path, sm.ClientState())
					suite.Require().NoError(err)
				}, false,
			},
			{
				"client", solomachine.CLIENT, func() {
					path := sm.GetClientStatePath(counterpartyClientIdentifier)
					data, err = solomachine.ClientStateDataBytes(cdc, path, sm.ClientState())
					suite.Require().NoError(err)
				}, true,
			},
			{
				"bad client (provides consensus state data)", solomachine.CLIENT, func() {
					path := sm.GetConsensusStatePath(counterpartyClientIdentifier, clienttypes.NewHeight(0, 5))
					data, err = solomachine.ConsensusStateDataBytes(cdc, path, sm.ConsensusState())
					suite.Require().NoError(err)
				}, false,
			},
			{
				"consensus", solomachine.CONSENSUS, func() {
					path := sm.GetConsensusStatePath(counterpartyClientIdentifier, clienttypes.NewHeight(0, 5))
					data, err = solomachine.ConsensusStateDataBytes(cdc, path, sm.ConsensusState())
					suite.Require().NoError(err)

				}, true,
			},
			{
				"bad consensus (provides client state data)", solomachine.CONSENSUS, func() {
					path := sm.GetClientStatePath(counterpartyClientIdentifier)
					data, err = solomachine.ClientStateDataBytes(cdc, path, sm.ClientState())
					suite.Require().NoError(err)
				}, false,
			},
			{
				"connection", solomachine.CONNECTION, func() {
					counterparty := connectiontypes.NewCounterparty("clientB", testConnectionID, *prefix)
					conn := connectiontypes.NewConnectionEnd(connectiontypes.OPEN, "clientA", counterparty, connectiontypes.ExportedVersionsToProto(connectiontypes.GetCompatibleVersions()), 0)
					path := sm.GetConnectionStatePath("connectionID")

					data, err = solomachine.ConnectionStateDataBytes(cdc, path, conn)
					suite.Require().NoError(err)

				}, true,
			},
			{
				"bad connection (uses channel data)", solomachine.CONNECTION, func() {
					counterparty := channeltypes.NewCounterparty(testPortID, testChannelID)
					ch := channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED, counterparty, []string{testConnectionID}, "1.0.0")
					path := sm.GetChannelStatePath("portID", "channelID")

					data, err = solomachine.ChannelStateDataBytes(cdc, path, ch)
					suite.Require().NoError(err)
				}, false,
			},
			{
				"channel", solomachine.CHANNEL, func() {
					counterparty := channeltypes.NewCounterparty(testPortID, testChannelID)
					ch := channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED, counterparty, []string{testConnectionID}, "1.0.0")
					path := sm.GetChannelStatePath("portID", "channelID")

					data, err = solomachine.ChannelStateDataBytes(cdc, path, ch)
					suite.Require().NoError(err)
				}, true,
			},
			{
				"bad channel (uses connection data)", solomachine.CHANNEL, func() {
					counterparty := connectiontypes.NewCounterparty("clientB", testConnectionID, *prefix)
					conn := connectiontypes.NewConnectionEnd(connectiontypes.OPEN, "clientA", counterparty, connectiontypes.ExportedVersionsToProto(connectiontypes.GetCompatibleVersions()), 0)
					path := sm.GetConnectionStatePath("connectionID")

					data, err = solomachine.ConnectionStateDataBytes(cdc, path, conn)
					suite.Require().NoError(err)

				}, false,
			},
			{
				"packet commitment", solomachine.PACKETCOMMITMENT, func() {
					commitment := []byte("packet commitment")
					path := sm.GetPacketCommitmentPath("portID", "channelID")

					data, err = solomachine.PacketCommitmentDataBytes(cdc, path, commitment)
					suite.Require().NoError(err)
				}, true,
			},
			{
				"bad packet commitment (uses next seq recv)", solomachine.PACKETCOMMITMENT, func() {
					path := sm.GetNextSequenceRecvPath("portID", "channelID")

					data, err = solomachine.NextSequenceRecvDataBytes(cdc, path, 10)
					suite.Require().NoError(err)
				}, false,
			},
			{
				"packet acknowledgement", solomachine.PACKETACKNOWLEDGEMENT, func() {
					commitment := []byte("packet acknowledgement")
					path := sm.GetPacketAcknowledgementPath("portID", "channelID")

					data, err = solomachine.PacketAcknowledgementDataBytes(cdc, path, commitment)
					suite.Require().NoError(err)
				}, true,
			},
			{
				"bad packet acknowledgement (uses next sequence recv)", solomachine.PACKETACKNOWLEDGEMENT, func() {
					path := sm.GetNextSequenceRecvPath("portID", "channelID")

					data, err = solomachine.NextSequenceRecvDataBytes(cdc, path, 10)
					suite.Require().NoError(err)
				}, false,
			},
			{
				"packet acknowledgement absence", solomachine.PACKETRECEIPTABSENCE, func() {
					path := sm.GetPacketReceiptPath("portID", "channelID")

					data, err = solomachine.PacketReceiptAbsenceDataBytes(cdc, path)
					suite.Require().NoError(err)
				}, true,
			},
			{
				"next sequence recv", solomachine.NEXTSEQUENCERECV, func() {
					path := sm.GetNextSequenceRecvPath("portID", "channelID")

					data, err = solomachine.NextSequenceRecvDataBytes(cdc, path, 10)
					suite.Require().NoError(err)
				}, true,
			},
			{
				"bad next sequence recv (uses packet commitment)", solomachine.NEXTSEQUENCERECV, func() {
					commitment := []byte("packet commitment")
					path := sm.GetPacketCommitmentPath("portID", "channelID")

					data, err = solomachine.PacketCommitmentDataBytes(cdc, path, commitment)
					suite.Require().NoError(err)
				}, false,
			},
		}

		for _, tc := range cases {
			tc := tc

			suite.Run(tc.name, func() {
				tc.malleate()

				data, err := solomachine.UnmarshalDataByType(cdc, tc.dataType, data)

				if tc.expPass {
					suite.Require().NoError(err)
					suite.Require().NotNil(data)
				} else {
					suite.Require().Error(err)
					suite.Require().Nil(data)
				}
			})
		}
	}

}
