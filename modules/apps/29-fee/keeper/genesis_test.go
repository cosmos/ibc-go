package keeper_test

import (
	"github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestInitGenesis() {
	packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

	genesisState := types.GenesisState{
		IdentifiedFees: []types.IdentifiedPacketFees{
			{
				PacketId: packetID,
				PacketFees: []types.PacketFee{
					{
						Fee:           types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee),
						RefundAddress: s.chainA.SenderAccount.GetAddress().String(),
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
		RegisteredPayees: []types.RegisteredPayee{
			{
				Relayer:   s.chainA.SenderAccount.GetAddress().String(),
				Payee:     s.chainB.SenderAccount.GetAddress().String(),
				ChannelId: ibctesting.FirstChannelID,
			},
		},
		RegisteredCounterpartyPayees: []types.RegisteredCounterpartyPayee{
			{
				Relayer:           s.chainA.SenderAccount.GetAddress().String(),
				CounterpartyPayee: s.chainB.SenderAccount.GetAddress().String(),
				ChannelId:         ibctesting.FirstChannelID,
			},
		},
	}

	s.chainA.GetSimApp().IBCFeeKeeper.InitGenesis(s.chainA.GetContext(), genesisState)

	// check fee
	feesInEscrow, found := s.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(s.chainA.GetContext(), packetID)
	s.Require().True(found)
	s.Require().Equal(genesisState.IdentifiedFees[0].PacketFees, feesInEscrow.PacketFees)

	// check fee is enabled
	isEnabled := s.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(s.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)
	s.Require().True(isEnabled)

	// check payee addresses
	payeeAddr, found := s.chainA.GetSimApp().IBCFeeKeeper.GetPayeeAddress(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress().String(), ibctesting.FirstChannelID)
	s.Require().True(found)
	s.Require().Equal(genesisState.RegisteredPayees[0].Payee, payeeAddr)

	// check relayers
	counterpartyPayeeAddr, found := s.chainA.GetSimApp().IBCFeeKeeper.GetCounterpartyPayeeAddress(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress().String(), ibctesting.FirstChannelID)
	s.Require().True(found)
	s.Require().Equal(genesisState.RegisteredCounterpartyPayees[0].CounterpartyPayee, counterpartyPayeeAddr)
}

func (s *KeeperTestSuite) TestExportGenesis() {
	// set fee enabled
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

	// setup & escrow the packet fee
	refundAcc := s.chainA.SenderAccount.GetAddress()
	packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

	packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})
	s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))

	// set payee address
	s.chainA.GetSimApp().IBCFeeKeeper.SetPayeeAddress(
		s.chainA.GetContext(),
		s.chainA.SenderAccount.GetAddress().String(),
		s.chainB.SenderAccount.GetAddress().String(),
		ibctesting.FirstChannelID,
	)

	// set counterparty payee address
	s.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(
		s.chainA.GetContext(),
		s.chainA.SenderAccount.GetAddress().String(),
		s.chainB.SenderAccount.GetAddress().String(),
		ibctesting.FirstChannelID,
	)

	// set forward relayer address
	s.chainA.GetSimApp().IBCFeeKeeper.SetRelayerAddressForAsyncAck(s.chainA.GetContext(), packetID, s.chainA.SenderAccount.GetAddress().String())

	// export genesis
	genesisState := s.chainA.GetSimApp().IBCFeeKeeper.ExportGenesis(s.chainA.GetContext())

	// check fee enabled
	s.Require().Equal(ibctesting.FirstChannelID, genesisState.FeeEnabledChannels[0].ChannelId)
	s.Require().Equal(ibctesting.MockFeePort, genesisState.FeeEnabledChannels[0].PortId)

	// check fee
	s.Require().Equal(packetID, genesisState.IdentifiedFees[0].PacketId)
	s.Require().Equal(fee, genesisState.IdentifiedFees[0].PacketFees[0].Fee)
	s.Require().Equal(refundAcc.String(), genesisState.IdentifiedFees[0].PacketFees[0].RefundAddress)
	s.Require().Equal([]string(nil), genesisState.IdentifiedFees[0].PacketFees[0].Relayers)

	// check forward relayer addresses
	s.Require().Equal(s.chainA.SenderAccount.GetAddress().String(), genesisState.ForwardRelayers[0].Address)
	s.Require().Equal(packetID, genesisState.ForwardRelayers[0].PacketId)

	// check payee addresses
	s.Require().Equal(s.chainA.SenderAccount.GetAddress().String(), genesisState.RegisteredPayees[0].Relayer)
	s.Require().Equal(s.chainB.SenderAccount.GetAddress().String(), genesisState.RegisteredPayees[0].Payee)
	s.Require().Equal(ibctesting.FirstChannelID, genesisState.RegisteredPayees[0].ChannelId)

	// check registered counterparty payee addresses
	s.Require().Equal(s.chainA.SenderAccount.GetAddress().String(), genesisState.RegisteredCounterpartyPayees[0].Relayer)
	s.Require().Equal(s.chainB.SenderAccount.GetAddress().String(), genesisState.RegisteredCounterpartyPayees[0].CounterpartyPayee)
	s.Require().Equal(ibctesting.FirstChannelID, genesisState.RegisteredCounterpartyPayees[0].ChannelId)
}
