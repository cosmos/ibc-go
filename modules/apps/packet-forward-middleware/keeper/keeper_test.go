package keeper_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"

	pfmtypes "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestWriteAcknowledgementForForwardedPacket() {
	fundAcc := func(ctx sdk.Context, bk bankkeeper.Keeper, acc sdk.AccAddress) {
		coins := sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10000000000))
		err := bk.MintCoins(ctx, "transfer", coins)
		s.Require().NoError(err)
		err = bk.SendCoinsFromModuleToAccount(ctx, "transfer", acc, coins)
		s.Require().NoError(err)
	}

	var expectedAckBz []byte

	tests := []struct {
		name          string
		ack           channeltypes.Acknowledgement
		malleate      func()
		nonRefundable bool
	}{
		{
			name:          "Ack success -> propagated to ics4 wrapper",
			ack:           channeltypes.NewResultAcknowledgement([]byte{1}),
			nonRefundable: false,
		},
		{
			name: "Ack error + Non refundable -> Asset moved to recoverable account then propagate ack to ics4 wrapper",
			ack:  channeltypes.NewErrorAcknowledgement(nil),
			malleate: func() {
				ack := channeltypes.NewErrorAcknowledgement(nil)
				ackResult := fmt.Sprintf("packet forward failed after point of no return: %s", ack.GetError())
				newAck := channeltypes.NewResultAcknowledgement([]byte(ackResult))
				expectedAckBz = channeltypes.CommitAcknowledgement(newAck.Acknowledgement())
			},
			nonRefundable: true,
		},
		{
			name:          "Ack error + Refundable -> Escrow coin then propagate ack to ics4 wrapper",
			ack:           channeltypes.NewErrorAcknowledgement(nil),
			nonRefundable: false,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()

			pathBC := ibctesting.NewTransferPath(s.chainB, s.chainC)
			pathBC.Setup()

			ctxB := s.chainB.GetContext()
			pfmKeeperB := s.chainB.GetSimApp().PFMKeeper

			ctxC := s.chainC.GetContext()
			pfmKeeperC := s.chainC.GetSimApp().PFMKeeper

			srcPacket := channeltypes.Packet{
				Data:               []byte{1},
				Sequence:           1,
				SourcePort:         pathBC.EndpointA.ChannelConfig.PortID,
				SourceChannel:      pathBC.EndpointA.ChannelID,
				DestinationPort:    pathBC.EndpointB.ChannelConfig.PortID,
				DestinationChannel: pathBC.EndpointB.ChannelID,
				TimeoutHeight: clienttypes.Height{
					RevisionNumber: 10,
					RevisionHeight: 100,
				},
				TimeoutTimestamp: 10101001,
			}

			retries := uint8(2)
			timeout := pfmtypes.Duration(1010101010)

			initialSender := s.chainA.SenderAccount.GetAddress()
			// Simulate an "Override Receiver" on destination chain.
			intermediateAcc := s.chainB.SenderAccounts[1].SenderAccount.GetAddress()
			finalReceiver := s.chainB.SenderAccount.GetAddress()

			metadata := pfmtypes.ForwardMetadata{
				Receiver: finalReceiver.String(),
				Port:     pathBC.EndpointA.ChannelConfig.PortID,
				Channel:  pathBC.EndpointA.ChannelID,
				Timeout:  timeout,
				Retries:  &retries,
				Next:     nil,
			}

			fundAcc(ctxB, s.chainB.GetSimApp().BankKeeper, intermediateAcc)

			err := pfmKeeperB.ForwardTransferPacket(ctxB, nil, srcPacket, initialSender.String(), intermediateAcc.String(), &metadata, ibctesting.TestCoin, 2, time.Duration(timeout), nil, tc.nonRefundable)
			s.Require().NoError(err)

			inflightPacket, err := pfmKeeperB.GetInflightPacket(ctxB, srcPacket)
			s.Require().NoError(err)

			token := transfertypes.NewToken(ibctesting.TestCoin.GetDenom(), ibctesting.DefaultCoinAmount.String())
			data := transfertypes.NewInternalTransferRepresentation(token, initialSender.String(), finalReceiver.String(), "")
			expectedAckBz = channeltypes.CommitAcknowledgement(tc.ack.Acknowledgement())
			if tc.malleate != nil {
				tc.malleate()
			}

			// Escrow on chainC
			escrow := transfertypes.GetEscrowAddress(srcPacket.SourcePort, srcPacket.SourceChannel)
			fundAcc(ctxC, s.chainC.GetSimApp().BankKeeper, escrow)

			err = pfmKeeperC.WriteAcknowledgementForForwardedPacket(ctxC, srcPacket, data, inflightPacket, tc.ack)
			s.Require().NoError(err)

			ackBZFromStore := s.chainC.GetAcknowledgement(srcPacket)
			s.Require().True(bytes.Equal(expectedAckBz, ackBZFromStore))
		})
	}
}

func (s *KeeperTestSuite) TestForwardTransferPacket() {
	s.SetupTest()
	path := ibctesting.NewTransferPath(s.chainA, s.chainB)
	path.Setup()

	s.chainA.GetSimApp().PFMKeeper.SetTransferKeeper(&transferMock{})
	ctx := s.chainA.GetContext()
	srcPacket := channeltypes.Packet{
		Data:               []byte{1},
		Sequence:           1,
		SourcePort:         path.EndpointA.ChannelConfig.PortID,
		SourceChannel:      path.EndpointA.ChannelID,
		DestinationPort:    path.EndpointB.ChannelConfig.PortID,
		DestinationChannel: path.EndpointB.ChannelID,
		TimeoutHeight: clienttypes.Height{
			RevisionNumber: 10,
			RevisionHeight: 100,
		},
		TimeoutTimestamp: 10101001,
	}

	retries := uint8(2)
	timeout := pfmtypes.Duration(1010101010)
	nonRefundable := false

	metadata := pfmtypes.ForwardMetadata{
		Receiver: "first-receiver",
		Port:     path.EndpointA.ChannelConfig.PortID,
		Channel:  path.EndpointA.ChannelID,
		Timeout:  timeout,
		Retries:  &retries,
		Next:     nil,
	}

	initialSender := s.chainA.SenderAccount.GetAddress()
	finalReceiver := s.chainB.SenderAccount.GetAddress()

	err := s.chainA.GetSimApp().PFMKeeper.ForwardTransferPacket(ctx, nil, srcPacket, initialSender.String(), finalReceiver.String(), &metadata, sdk.NewInt64Coin("denom", 1000), 2, time.Duration(timeout), nil, nonRefundable)
	s.Require().NoError(err)

	// Get the inflight packer
	inflightPacket, err := s.chainA.GetSimApp().PFMKeeper.GetInflightPacket(ctx, srcPacket)
	s.Require().NoError(err)

	s.Require().Equal(inflightPacket.RetriesRemaining, int32(retries))

	// Call the same function again with inflight packet. Num retries should decrease.
	err = s.chainA.GetSimApp().PFMKeeper.ForwardTransferPacket(ctx, inflightPacket, srcPacket, initialSender.String(), finalReceiver.String(), &metadata, sdk.NewInt64Coin("denom", 1000), 2, time.Duration(timeout), nil, nonRefundable)
	s.Require().NoError(err)

	// Get the inflight packer
	inflightPacket2, err := s.chainA.GetSimApp().PFMKeeper.GetInflightPacket(ctx, srcPacket)
	s.Require().NoError(err)

	s.Require().Equal(inflightPacket.RetriesRemaining, inflightPacket2.RetriesRemaining)
	s.Require().Equal(int32(retries-1), inflightPacket.RetriesRemaining)
}

type transferMock struct{}

func (*transferMock) Transfer(_ context.Context, _ *transfertypes.MsgTransfer) (*transfertypes.MsgTransferResponse, error) {
	return &transfertypes.MsgTransferResponse{
		Sequence: 1,
	}, nil
}

func (*transferMock) GetDenom(_ sdk.Context, _ cmtbytes.HexBytes) (transfertypes.Denom, bool) {
	return transfertypes.Denom{}, false
}

func (*transferMock) GetTotalEscrowForDenom(ctx sdk.Context, denom string) sdk.Coin {
	return sdk.Coin{}
}

func (*transferMock) SetTotalEscrowForDenom(ctx sdk.Context, coin sdk.Coin) {
}

func (*transferMock) DenomPathFromHash(ctx sdk.Context, ibcDenom string) (string, error) {
	return "", nil
}

func (*transferMock) GetPort(ctx sdk.Context) string {
	return ""
}
