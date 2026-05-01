package keeper_test

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	pfmkeeper "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/keeper"
	"github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/migrations/v3"
	"github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/migrations/v4/legacy"
	pfmtypes "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

func (s *KeeperTestSuite) TestMigrate2to3() {
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

func (s *KeeperTestSuite) TestMigrate3to4() {
	var addLegacyPacket func(seq uint64, nonrefundable bool) string
	var setRawPacket func(channelID string, seq uint64, rawBz []byte) string

	tests := []struct {
		name        string
		malleate    func()
		expectedErr string
	}{
		{
			name: "success: empty store",
			malleate: func() {
			},
			expectedErr: "",
		},
		{
			name: "success: one refundable in-flight packet",
			malleate: func() {
				addLegacyPacket(1, false)
			},
			expectedErr: "",
		},
		{
			name: "success: many refundable in-flight packets",
			malleate: func() {
				addLegacyPacket(1, false)
				addLegacyPacket(2, false)
				addLegacyPacket(3, false)
				addLegacyPacket(4, false)
			},
			expectedErr: "",
		},
		{
			name: "failure: one nonrefundable in-flight packet",
			malleate: func() {
				addLegacyPacket(1, true)
			},
			expectedErr: string(pfmtypes.RefundPacketKey("channel-1", "transfer", 1)),
		},
		{
			name: "failure: one nonrefundable and many refundable in-flight packets",
			malleate: func() {
				addLegacyPacket(1, true)
				addLegacyPacket(2, false)
				addLegacyPacket(3, false)
				addLegacyPacket(4, false)
			},
			expectedErr: string(pfmtypes.RefundPacketKey("channel-1", "transfer", 1)),
		},
		{
			name: "failure: invalid legacy bytes in store",
			malleate: func() {
				setRawPacket("channel-1", 1, []byte{0xff, 0xff, 0xff})
			},
			expectedErr: "failed to unmarshal legacy in-flight packet",
		},
		{
			name: "failure: invalid packet after conversion",
			malleate: func() {
				channelID := "channel-1"
				portID := "transfer"

				legacyPacket := legacy.InFlightPacket{
					PacketData:             []byte{1, 2, 3},
					OriginalSenderAddress:  s.chainA.SenderAccount.GetAddress().String(),
					RefundChannelId:        "channel-refund",
					RefundPortId:           portID,
					PacketSrcChannelId:     channelID,
					PacketSrcPortId:        "",
					PacketTimeoutTimestamp: 100,
					PacketTimeoutHeight:    "0-10",
					RefundSequence:         1,
					RetriesRemaining:       1,
					Timeout:                1000,
					Nonrefundable:          false,
				}

				rawBz, err := legacyPacket.Marshal()
				s.Require().NoError(err)

				setRawPacket(channelID, 1, rawBz)
			},
			expectedErr: "invalid in-flight packet found during migration",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()

			ctx := s.chainA.GetContext()
			keeper := s.chainA.GetSimApp().PFMKeeper
			storeService := runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(pfmtypes.StoreKey))
			store := storeService.OpenKVStore(ctx)

			addLegacyPacket = func(seq uint64, nonrefundable bool) string {
				channelID := fmt.Sprintf("channel-%d", seq)
				portID := "transfer"

				legacyPacket := legacy.InFlightPacket{
					PacketData:             []byte{1, byte(seq)},
					OriginalSenderAddress:  s.chainA.SenderAccount.GetAddress().String(),
					RefundChannelId:        "channel-refund",
					RefundPortId:           portID,
					PacketSrcChannelId:     channelID,
					PacketSrcPortId:        portID,
					PacketTimeoutTimestamp: 100,
					PacketTimeoutHeight:    "0-10",
					RefundSequence:         seq,
					RetriesRemaining:       1,
					Timeout:                1000,
					Nonrefundable:          nonrefundable,
				}

				rawBz, err := legacyPacket.Marshal()
				s.Require().NoError(err)

				key := pfmtypes.RefundPacketKey(channelID, portID, seq)
				err = store.Set(key, rawBz)
				s.Require().NoError(err)

				return string(key)
			}

			setRawPacket = func(channelID string, seq uint64, rawBz []byte) string {
				key := pfmtypes.RefundPacketKey(channelID, "transfer", seq)
				err := store.Set(key, rawBz)
				s.Require().NoError(err)

				return string(key)
			}

			tc.malleate()

			migrator := pfmkeeper.NewMigrator(keeper)
			err := migrator.Migrate3to4(ctx)

			if tc.expectedErr != "" {
				s.Require().ErrorContains(err, tc.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}
