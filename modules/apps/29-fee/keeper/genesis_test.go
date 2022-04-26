package keeper_test

import (
	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	// build PacketId & Fee
	refundAcc := suite.chainA.SenderAccount.GetAddress()
	packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
	fee := types.Fee{
		RecvFee:    defaultRecvFee,
		AckFee:     defaultAckFee,
		TimeoutFee: defaultTimeoutFee,
	}

	// relayer addresses
	sender := suite.chainA.SenderAccount.GetAddress().String()
	counterparty := suite.chainB.SenderAccount.GetAddress().String()

	genesisState := types.GenesisState{
		IdentifiedFees: []types.IdentifiedPacketFees{
			{
				PacketId: packetID,
				PacketFees: []types.PacketFee{
					{
						Fee:           fee,
						RefundAddress: refundAcc.String(),
						Relayers:      nil,
					},
				},
			},
		},
		FeeEnabledChannels: []types.FeeEnabledChannel{
			{
				PortId:    ibctesting.MockFeePort,
				ChannelId: ibctesting.FirstChannelID,
			},
		},
		RegisteredRelayers: []types.RegisteredRelayerAddress{
			{
				Address:             sender,
				CounterpartyAddress: counterparty,
				ChannelId:           ibctesting.FirstChannelID,
			},
		},
	}

	suite.chainA.GetSimApp().IBCFeeKeeper.InitGenesis(suite.chainA.GetContext(), genesisState)

	// check fee
	feesInEscrow, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(suite.chainA.GetContext(), packetID)
	suite.Require().True(found)
	suite.Require().Equal(genesisState.IdentifiedFees[0].PacketFees, feesInEscrow.PacketFees)

	// check fee is enabled
	isEnabled := suite.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)
	suite.Require().True(isEnabled)

	// check relayers
	addr, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetCounterpartyAddress(suite.chainA.GetContext(), sender, ibctesting.FirstChannelID)
	suite.Require().True(found)
	suite.Require().Equal(genesisState.RegisteredRelayers[0].CounterpartyAddress, addr)
}

func (suite *KeeperTestSuite) TestExportGenesis() {
	// set fee enabled
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

	// setup & escrow the packet fee
	refundAcc := suite.chainA.SenderAccount.GetAddress()
	packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
	fee := types.Fee{
		RecvFee:    defaultRecvFee,
		AckFee:     defaultAckFee,
		TimeoutFee: defaultTimeoutFee,
	}

	packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))

	// relayer addresses
	sender := suite.chainA.SenderAccount.GetAddress().String()
	counterparty := suite.chainB.SenderAccount.GetAddress().String()
	// set counterparty address
	suite.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyAddress(suite.chainA.GetContext(), sender, counterparty, ibctesting.FirstChannelID)

	// set forward relayer address
	suite.chainA.GetSimApp().IBCFeeKeeper.SetRelayerAddressForAsyncAck(suite.chainA.GetContext(), packetID, sender)

	// export genesis
	genesisState := suite.chainA.GetSimApp().IBCFeeKeeper.ExportGenesis(suite.chainA.GetContext())

	// check fee enabled
	suite.Require().Equal(ibctesting.FirstChannelID, genesisState.FeeEnabledChannels[0].ChannelId)
	suite.Require().Equal(ibctesting.MockFeePort, genesisState.FeeEnabledChannels[0].PortId)

	// check fee
	suite.Require().Equal(packetID, genesisState.IdentifiedFees[0].PacketId)
	suite.Require().Equal(fee, genesisState.IdentifiedFees[0].PacketFees[0].Fee)
	suite.Require().Equal(refundAcc.String(), genesisState.IdentifiedFees[0].PacketFees[0].RefundAddress)
	suite.Require().Equal([]string(nil), genesisState.IdentifiedFees[0].PacketFees[0].Relayers)

	// check registered relayer addresses
	suite.Require().Equal(sender, genesisState.RegisteredRelayers[0].Address)
	suite.Require().Equal(counterparty, genesisState.RegisteredRelayers[0].CounterpartyAddress)

	// check registered relayer addresses
	suite.Require().Equal(sender, genesisState.ForwardRelayers[0].Address)
	suite.Require().Equal(packetID, genesisState.ForwardRelayers[0].PacketId)
}
