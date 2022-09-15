package ibc_hooks_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibchooks "github.com/cosmos/ibc-go/v5/modules/apps/hooks"
	transfertypes "github.com/cosmos/ibc-go/v5/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v5/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
	ibcmock "github.com/cosmos/ibc-go/v5/testing/mock"
	"github.com/stretchr/testify/suite"
	"testing"
)

var _ ibchooks.IBCAppHooks = TestOverrideHooks{}
var _ ibchooks.IBCAppHooks = TestBeforeAfterHooks{}

type Status struct {
	OverrideRan bool
	BeforeRan   bool
	AfterRan    bool
}

type TestOverrideHooks struct{ Status *Status }

func (t TestOverrideHooks) OnRecvPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	t.Status.OverrideRan = true
	ack := im.App.OnRecvPacket(ctx, packet, relayer)
	return ack
}

type TestBeforeAfterHooks struct{ Status *Status }

func (t TestBeforeAfterHooks) OnRecvPacketBeforeHook(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) {
	t.Status.BeforeRan = true
}
func (t TestBeforeAfterHooks) OnRecvPacketAfterHook(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress, ack ibcexported.Acknowledgement) {
	t.Status.AfterRan = true
}

type HooksTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain

	path     *ibctesting.Path
	pathAToC *ibctesting.Path
}

func (suite *HooksTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))
}

func TestIBCHooksTestSuite(t *testing.T) {
	suite.Run(t, new(HooksTestSuite))
}

func (suite *HooksTestSuite) CreateMockPacket() channeltypes.Packet {
	return channeltypes.NewPacket(
		ibcmock.MockPacketData,
		suite.chainA.SenderAccount.GetSequence(),
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(0, 100),
		0,
	)
}

func NewTransferPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointA.ChannelConfig.Version = transfertypes.Version
	path.EndpointB.ChannelConfig.Version = transfertypes.Version

	return path
}

func (suite *HooksTestSuite) TestOnRecvPacketHooks() {
	var (
		trace    transfertypes.DenomTrace
		amount   math.Int
		receiver string
		status   Status
	)

	testCases := []struct {
		msg          string
		malleate     func(*Status)
		recvIsSource bool // the receiving chain is the source of the coin originally
		expPass      bool
	}{
		{"override", func(status *Status) {
			suite.chainB.GetSimApp().HooksMiddleware.Hooks = TestOverrideHooks{Status: status}
		}, true, true},
		{"before and after", func(status *Status) {
			suite.chainB.GetSimApp().HooksMiddleware.Hooks = TestBeforeAfterHooks{Status: status}
		}, false, true},
		{"before and after with error", func(status *Status) {
			suite.chainB.GetSimApp().HooksMiddleware.Hooks = TestBeforeAfterHooks{Status: status}
		}, false, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset

			path := NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)
			receiver = suite.chainB.SenderAccount.GetAddress().String() // must be explicitly changed in malleate
			status = Status{}

			amount = sdk.NewInt(100) // must be explicitly changed in malleate
			seq := uint64(1)

			if tc.recvIsSource {
				// send coin from chainB to chainA, receive them, acknowledge them, and send back to chainB
				coinFromBToA := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))
				transferMsg := transfertypes.NewMsgTransfer(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, coinFromBToA, suite.chainB.SenderAccount.GetAddress().String(), suite.chainA.SenderAccount.GetAddress().String(), clienttypes.NewHeight(1, 110), 0)
				res, err := suite.chainB.SendMsgs(transferMsg)
				suite.Require().NoError(err) // message committed

				packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
				suite.Require().NoError(err)

				err = path.RelayPacket(packet)
				suite.Require().NoError(err) // relay committed

				seq++

				// NOTE: trace must be explicitly changed in malleate to test invalid cases
				trace = transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.DefaultBondDenom))
			} else {
				trace = transfertypes.ParseDenomTrace(sdk.DefaultBondDenom)
			}

			// send coin from chainA to chainB
			transferMsg := transfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.NewCoin(trace.IBCDenom(), amount), suite.chainA.SenderAccount.GetAddress().String(), receiver, clienttypes.NewHeight(1, 110), 0)
			_, err := suite.chainA.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			tc.malleate(&status)

			data := transfertypes.NewFungibleTokenPacketData(trace.GetFullDenomPath(), amount.String(), suite.chainA.SenderAccount.GetAddress().String(), receiver)
			packet := channeltypes.NewPacket(data.GetBytes(), seq, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)

			ack := suite.chainB.GetSimApp().HooksMiddleware.OnRecvPacket(suite.chainB.GetContext(), packet, suite.chainC.SenderAccount.GetAddress())

			if tc.expPass {
				suite.Require().True(ack.Success())
			} else {
				suite.Require().False(ack.Success())
			}

			if _, ok := suite.chainB.GetSimApp().HooksMiddleware.Hooks.(TestBeforeAfterHooks); ok {
				suite.Require().False(status.OverrideRan)
				suite.Require().True(status.BeforeRan)
				suite.Require().True(status.AfterRan)
			}
			if _, ok := suite.chainB.GetSimApp().HooksMiddleware.Hooks.(TestOverrideHooks); ok {
				suite.Require().True(status.OverrideRan)
				suite.Require().False(status.BeforeRan)
				suite.Require().False(status.AfterRan)
			}
		})
	}
}
