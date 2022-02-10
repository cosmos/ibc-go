package keeper_test

import (
	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	suite.SetupTest()

	// build PacketId & Fee
	refundAcc := suite.chainA.SenderAccount.GetAddress()
	packetId := channeltypes.NewPacketId(
		ibctesting.FirstChannelID,
		transfertypes.PortID,
		uint64(1),
	)
	fee := types.Fee{
		defaultReceiveFee,
		defaultAckFee,
		defaultTimeoutFee,
	}

	// relayer addresses
	sender := suite.chainA.SenderAccount.GetAddress().String()
	counterparty := suite.chainB.SenderAccount.GetAddress().String()

	genesisState := types.GenesisState{
		IdentifiedFees: []types.IdentifiedPacketFee{
			{
				PacketId:      packetId,
				Fee:           fee,
				RefundAddress: refundAcc.String(),
				Relayers:      nil,
			},
		},
		FeeEnabledChannels: []*types.FeeEnabledChannel{
			{
				PortId:    transfertypes.PortID,
				ChannelId: ibctesting.FirstChannelID,
			},
		},
		RegisteredRelayers: []*types.RegisteredRelayerAddress{
			{
				Address:             sender,
				CounterpartyAddress: counterparty,
			},
		},
	}

	suite.chainA.GetSimApp().IBCFeeKeeper.InitGenesis(suite.chainA.GetContext(), genesisState)

	// check fee
	identifiedFee, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeInEscrow(suite.chainA.GetContext(), packetId)
	suite.Require().True(found)
	suite.Require().Equal(genesisState.IdentifiedFees[0], identifiedFee)

	// check fee is enabled
	isEnabled := suite.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainA.GetContext(), transfertypes.PortID, ibctesting.FirstChannelID)
	suite.Require().True(isEnabled)

	// check relayers
	addr, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetCounterpartyAddress(suite.chainA.GetContext(), sender)
	suite.Require().True(found)
	suite.Require().Equal(genesisState.RegisteredRelayers[0].CounterpartyAddress, addr)
}

func (suite *KeeperTestSuite) TestExportGenesis() {
	suite.SetupTest()
	// set fee enabled
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), transfertypes.PortID, ibctesting.FirstChannelID)

	// setup & escrow the packet fee
	refundAcc := suite.chainA.SenderAccount.GetAddress()
	packetId := channeltypes.NewPacketId(
		ibctesting.FirstChannelID,
		transfertypes.PortID,
		uint64(1),
	)
	fee := types.Fee{
		defaultReceiveFee,
		defaultAckFee,
		defaultTimeoutFee,
	}
	identifiedPacketFee := types.NewIdentifiedPacketFee(packetId, fee, refundAcc.String(), []string{})
	err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedPacketFee)
	suite.Require().NoError(err)

	// relayer addresses
	sender := suite.chainA.SenderAccount.GetAddress().String()
	counterparty := suite.chainB.SenderAccount.GetAddress().String()
	// set counterparty address
	suite.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyAddress(suite.chainA.GetContext(), sender, counterparty)

	// set forward relayer address
	suite.chainA.GetSimApp().IBCFeeKeeper.SetForwardRelayerAddress(suite.chainA.GetContext(), packetId, sender)

	// export genesis
	genesisState := suite.chainA.GetSimApp().IBCFeeKeeper.ExportGenesis(suite.chainA.GetContext())

	// check fee enabled
	suite.Require().Equal(ibctesting.FirstChannelID, genesisState.FeeEnabledChannels[0].ChannelId)
	suite.Require().Equal(transfertypes.PortID, genesisState.FeeEnabledChannels[0].PortId)

	// check fee
	suite.Require().Equal(packetId, genesisState.IdentifiedFees[0].PacketId)
	suite.Require().Equal(fee, genesisState.IdentifiedFees[0].Fee)
	suite.Require().Equal(refundAcc.String(), genesisState.IdentifiedFees[0].RefundAddress)
	suite.Require().Equal([]string(nil), genesisState.IdentifiedFees[0].Relayers)

	// check registered relayer addresses
	suite.Require().Equal(sender, genesisState.RegisteredRelayers[0].Address)
	suite.Require().Equal(counterparty, genesisState.RegisteredRelayers[0].CounterpartyAddress)

	// check registered relayer addresses
	suite.Require().Equal(sender, genesisState.ForwardRelayers[0].Address)
	suite.Require().Equal(packetId, genesisState.ForwardRelayers[0].PacketId)
}
