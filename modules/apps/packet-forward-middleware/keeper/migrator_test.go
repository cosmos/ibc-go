package keeper_test

import (
	"math/rand"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	pfmkeeper "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/migrations/v3"
	pfmtypes "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestMigrator() {
	retries := uint8(2)
	var (
		accA, accB, accC, port string
		firstHopMetadata       *pfmtypes.PacketMetadata
		err                    error
		nextMemo               string
		pathAB, pathBC         *ibctesting.Path
	)

	tests := []struct {
		name        string
		malleate    func()
		shouldEmpty bool
	}{
		{
			name: "A -> B -> C. A and B escrowed",
			malleate: func() {
				firstHopMetadata = &pfmtypes.PacketMetadata{
					Forward: pfmtypes.ForwardMetadata{
						Receiver: accC,
						Port:     port,
						Channel:  pathBC.EndpointA.ChannelID,
						Timeout:  time.Duration(100000000000),
						Retries:  &retries,
					},
				}
				nextMemo, err = firstHopMetadata.ToMemo()
				s.Require().NoError(err)
			},
			shouldEmpty: false,
		},
		{
			name: "A -> B -> A. Everything unescrowed",
			malleate: func() {
				firstHopMetadata = &pfmtypes.PacketMetadata{
					Forward: pfmtypes.ForwardMetadata{
						Receiver: accA,
						Port:     port,
						Channel:  pathAB.EndpointB.ChannelID,
						Timeout:  time.Duration(100000000000),
						Retries:  &retries,
					},
				}
				nextMemo, err = firstHopMetadata.ToMemo()
				s.Require().NoError(err)
			},
			shouldEmpty: true,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()

			accA = s.chainA.SenderAccount.GetAddress().String()
			accB = s.chainB.SenderAccount.GetAddress().String()
			accC = s.chainC.SenderAccount.GetAddress().String()

			pathAB = ibctesting.NewTransferPath(s.chainA, s.chainB)
			pathAB.Setup()

			pathBC = ibctesting.NewTransferPath(s.chainB, s.chainC)
			pathBC.Setup()

			transferKeeperA := s.chainA.GetSimApp().TransferKeeper
			transferKeeperB := s.chainB.GetSimApp().TransferKeeper

			port = pathBC.EndpointA.ChannelConfig.PortID

			ctxA := s.chainA.GetContext()
			ctxB := s.chainB.GetContext()

			denomA := transfertypes.Denom{Base: sdk.DefaultBondDenom}
			randSendAmt := int64(rand.Intn(10000000000) + 1000000)
			sendCoin := sdk.NewInt64Coin(denomA.IBCDenom(), randSendAmt)

			tc.malleate() // Hammer time!!!

			transferMsg := transfertypes.NewMsgTransfer(port, pathAB.EndpointA.ChannelID, sendCoin, accA, accB, s.chainB.GetTimeoutHeight(), 0, nextMemo)
			result, err := s.chainA.SendMsgs(transferMsg)
			s.Require().NoError(err)

			// Transfer escrowed on chainA and sent amount to chainB
			totalEscrowA, _ := v3.TotalEscrow(ctxA, s.chainA.GetSimApp().BankKeeper, s.chainA.App.GetIBCKeeper().ChannelKeeper, port)
			s.Require().Equal(randSendAmt, totalEscrowA[0].Amount.Int64())

			// ChainB has no escrow until the  packet is relayed.
			totalEscrowB, _ := v3.TotalEscrow(ctxB, s.chainB.GetSimApp().BankKeeper, s.chainB.App.GetIBCKeeper().ChannelKeeper, port)
			s.Require().Empty(totalEscrowB)

			packet, err := ibctesting.ParseV1PacketFromEvents(result.Events)
			s.Require().NoError(err)

			err = pathAB.RelayPacket(packet)
			s.Require().ErrorContains(err, "acknowledgement event attribute not found")

			// After the relay, we have amount escrowed on chainB
			totalEscrowA, _ = v3.TotalEscrow(ctxA, s.chainA.GetSimApp().BankKeeper, s.chainA.App.GetIBCKeeper().ChannelKeeper, port)
			s.Require().Equal(randSendAmt, totalEscrowA[0].Amount.Int64())

			totalEscrowB, _ = v3.TotalEscrow(ctxB, s.chainB.GetSimApp().BankKeeper, s.chainB.App.GetIBCKeeper().ChannelKeeper, port)
			if tc.shouldEmpty {
				s.Require().Empty(totalEscrowB)
			} else {
				s.Require().Equal(randSendAmt, totalEscrowB[0].Amount.Int64())
			}

			// Artificially set escrow balance to 0. So that we can show that after the migration, balances are restored.
			transferKeeperA.SetTotalEscrowForDenom(ctxA, sdk.NewInt64Coin(totalEscrowA[0].Denom, 0))
			if !tc.shouldEmpty {
				transferKeeperB.SetTotalEscrowForDenom(ctxB, sdk.NewInt64Coin(totalEscrowB[0].Denom, 0))
			}

			// Run the migration
			migratorA := pfmkeeper.NewMigrator(s.chainA.GetSimApp().PFMKeeper)
			err = migratorA.Migrate2to3(ctxA)
			s.Require().NoError(err)

			migratorB := pfmkeeper.NewMigrator(s.chainB.GetSimApp().PFMKeeper)
			err = migratorB.Migrate2to3(ctxB)
			s.Require().NoError(err)

			denomEscrowA := transferKeeperA.GetTotalEscrowForDenom(ctxA, totalEscrowA[0].Denom)
			s.Require().Equal(randSendAmt, denomEscrowA.Amount.Int64())

			if !tc.shouldEmpty {
				denomEscrowB := transferKeeperB.GetTotalEscrowForDenom(ctxB, totalEscrowB[0].Denom)
				s.Require().Equal(randSendAmt, denomEscrowB.Amount.Int64())
			}
		})
	}
}
