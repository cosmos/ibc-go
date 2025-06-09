package keeper_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	tests := []struct {
		name          string
		ackFn         func() (channeltypes.Acknowledgement, []byte)
		nonRefundable bool
	}{
		{
			name: "Ack success -> propagated to ics4 wrapper",
			ackFn: func() (channeltypes.Acknowledgement, []byte) {
				ack := channeltypes.NewResultAcknowledgement([]byte{1})
				ackBz := channeltypes.CommitAcknowledgement(ack.Acknowledgement())
				return ack, ackBz
			},
			nonRefundable: false,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()
			pathAB := ibctesting.NewTransferPath(s.chainA, s.chainB)
			pathAB.Setup()

			ctxA := s.chainA.GetContext()
			pfmKeeperA := s.chainA.GetSimApp().PFMKeeper
			pfmKeeperA.SetTransferKeeper(&stabTransfer{})

			ctxB := s.chainB.GetContext()
			pfmKeeperB := s.chainB.GetSimApp().PFMKeeper

			srcPacket := channeltypes.Packet{
				Data:               []byte{1},
				Sequence:           1,
				SourcePort:         pathAB.EndpointA.ChannelConfig.PortID,
				SourceChannel:      pathAB.EndpointA.ChannelID,
				DestinationPort:    pathAB.EndpointB.ChannelConfig.PortID,
				DestinationChannel: pathAB.EndpointB.ChannelID,
				TimeoutHeight: clienttypes.Height{
					RevisionNumber: 10,
					RevisionHeight: 100,
				},
				TimeoutTimestamp: 10101001,
			}

			retries := uint8(2)
			timeout := pfmtypes.Duration(1010101010)

			metadata := pfmtypes.ForwardMetadata{
				Receiver: "first-receiver",
				Port:     pathAB.EndpointA.ChannelConfig.PortID,
				Channel:  pathAB.EndpointA.ChannelID,
				Timeout:  timeout,
				Retries:  &retries,
				Next:     nil,
			}

			initialSender := s.chainA.SenderAccount.GetAddress()
			finalReceiver := s.chainB.SenderAccount.GetAddress()

			err := s.chainA.GetSimApp().BankKeeper.MintCoins(ctxA, "transfer", sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10000000000)))
			s.Require().NoError(err)
			err = s.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(ctxA, "transfer", initialSender, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10000000000)))
			s.Require().NoError(err)

			err = pfmKeeperA.ForwardTransferPacket(ctxA, nil, srcPacket, initialSender.String(), finalReceiver.String(), &metadata, ibctesting.TestCoin, 2, time.Duration(timeout), nil, tc.nonRefundable)
			s.Require().NoError(err)

			// Get the inflight packer
			inflightPacket, err := pfmKeeperA.GetInflightPacket(ctxA, srcPacket)
			s.Require().NoError(err)

			ack, ackBZ := tc.ackFn()
			token := transfertypes.NewFungibleTokenPacketData(ibctesting.TestCoin.GetDenom(), ibctesting.DefaultCoinAmount.String(), initialSender.String(), finalReceiver.String(), "")
			err = pfmKeeperB.WriteAcknowledgementForForwardedPacket(ctxB, srcPacket, token, inflightPacket, ack)
			s.Require().NoError(err)

			ackBZFromStore := s.chainB.GetAcknowledgement(srcPacket)
			s.Require().True(bytes.Equal(ackBZ, ackBZFromStore))
		})
	}
}

func (s *KeeperTestSuite) TestForwardTransferPacket() {
	s.SetupTest()
	path := ibctesting.NewTransferPath(s.chainA, s.chainB)
	path.Setup()

	s.chainA.GetSimApp().PFMKeeper.SetTransferKeeper(&stabTransfer{})
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

type stabTransfer struct{}

func (t *stabTransfer) Transfer(_ context.Context, _ *transfertypes.MsgTransfer) (*transfertypes.MsgTransferResponse, error) {
	return &transfertypes.MsgTransferResponse{
		Sequence: 1,
	}, nil
}

func (t *stabTransfer) GetDenom(_ sdk.Context, _ cmtbytes.HexBytes) (transfertypes.Denom, bool) {
	return transfertypes.Denom{}, false
}

func (t *stabTransfer) GetTotalEscrowForDenom(ctx sdk.Context, denom string) sdk.Coin {
	return sdk.Coin{}
}

func (t *stabTransfer) SetTotalEscrowForDenom(ctx sdk.Context, coin sdk.Coin) {
}

func (t *stabTransfer) DenomPathFromHash(ctx sdk.Context, ibcDenom string) (string, error) {
	return "", nil
}

func (t *stabTransfer) GetPort(ctx sdk.Context) string {
	return ""
}
