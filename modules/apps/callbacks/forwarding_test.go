package ibccallbacks_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/callbacks/testing/simapp"
	callbacktypes "github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func init() {
	ibctesting.DefaultTestingAppInit = SetupTestingApp
}

type CallbacksForwardingTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain

	pathAtoB *ibctesting.Path
	pathBtoC *ibctesting.Path
}

// setupChains sets up a coordinator with 3 test chains.
func (s *CallbacksForwardingTestSuite) setupChains() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))

	s.pathAtoB = ibctesting.NewTransferPath(s.chainA, s.chainB)
	s.pathBtoC = ibctesting.NewTransferPath(s.chainB, s.chainC)
}

func (s *CallbacksForwardingTestSuite) SetupTest() {
	s.setupChains()

	s.pathAtoB.Setup()
	s.pathBtoC.Setup()
}

func TestIBCCallbacksForwardingTestsuite(t *testing.T) {
	suite.Run(t, new(CallbacksForwardingTestSuite))
}

// TestForwardingWithMemoCallback tests that, when forwarding a packet with memo from A to B to C,
// the callback is executed only on the final hop.
// NOTE: this does not test the full forwarding behaviour (assert on amounts, packets, acks etc)
// as this is covered in other forwarding tests.
func (suite *CallbacksForwardingTestSuite) TestForwardingWithMemoCallback() {
	testCases := []struct {
		name                   string
		testMemo               string
		expCallbackMapOnChainB map[callbacktypes.CallbackType]int
		expCallbackMapOnChainC map[callbacktypes.CallbackType]int
	}{
		{
			name:                   "no memo",
			expCallbackMapOnChainB: map[callbacktypes.CallbackType]int{},
			expCallbackMapOnChainC: map[callbacktypes.CallbackType]int{},
		},
		{
			name:                   "recv callback",
			testMemo:               fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.SuccessContract),
			expCallbackMapOnChainB: map[callbacktypes.CallbackType]int{},
			expCallbackMapOnChainC: map[callbacktypes.CallbackType]int{callbacktypes.CallbackTypeReceivePacket: 1},
		},
		{
			name:                   "ack callback",
			testMemo:               fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
			expCallbackMapOnChainB: map[callbacktypes.CallbackType]int{callbacktypes.CallbackTypeAcknowledgementPacket: 1},
			expCallbackMapOnChainC: map[callbacktypes.CallbackType]int{},
		},
		{
			name:                   "ack callback with low gas (error)",
			testMemo:               fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.OogErrorContract),
			expCallbackMapOnChainB: map[callbacktypes.CallbackType]int{callbacktypes.CallbackTypeAcknowledgementPacket: 1},
			expCallbackMapOnChainC: map[callbacktypes.CallbackType]int{},
		},
		{
			name:                   "recv callback with low gas (error)",
			testMemo:               fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.OogErrorContract),
			expCallbackMapOnChainB: map[callbacktypes.CallbackType]int{},
			expCallbackMapOnChainC: map[callbacktypes.CallbackType]int{callbacktypes.CallbackTypeReceivePacket: 1},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			coinOnA := ibctesting.TestCoin
			sender := suite.chainA.SenderAccounts[0].SenderAccount
			receiver := suite.chainC.SenderAccounts[0].SenderAccount
			forwarding := types.NewForwarding(false, types.NewHop(
				suite.pathBtoC.EndpointA.ChannelConfig.PortID,
				suite.pathBtoC.EndpointA.ChannelID,
			))
			successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

			transferMsg := types.NewMsgTransfer(
				suite.pathAtoB.EndpointA.ChannelConfig.PortID,
				suite.pathAtoB.EndpointA.ChannelID,
				sdk.NewCoins(coinOnA),
				sender.GetAddress().String(),
				receiver.GetAddress().String(),
				clienttypes.ZeroHeight(),
				suite.chainA.GetTimeoutTimestamp(),
				tc.testMemo,
				forwarding,
			)

			result, err := suite.chainA.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			// parse the packet from result events and recv packet on chainB
			packetFromAtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
			suite.Require().NoError(err)
			suite.Require().NotNil(packetFromAtoB)

			err = suite.pathAtoB.EndpointB.UpdateClient()
			suite.Require().NoError(err)

			result, err = suite.pathAtoB.EndpointB.RecvPacketWithResult(packetFromAtoB)
			suite.Require().NoError(err)
			suite.Require().NotNil(result)

			packetFromBtoC, err := ibctesting.ParsePacketFromEvents(result.Events)
			suite.Require().NoError(err)
			suite.Require().NotNil(packetFromBtoC)

			err = suite.pathBtoC.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			err = suite.pathBtoC.EndpointB.UpdateClient()
			suite.Require().NoError(err)

			result, err = suite.pathBtoC.EndpointB.RecvPacketWithResult(packetFromBtoC)
			suite.Require().NoError(err)
			suite.Require().NotNil(result)

			packetOnC, err := ibctesting.ParseRecvPacketFromEvents(result.Events)
			suite.Require().NoError(err)
			suite.Require().NotNil(packetOnC)

			// Ack back to B
			err = suite.pathBtoC.EndpointB.UpdateClient()
			suite.Require().NoError(err)

			err = suite.pathBtoC.EndpointA.AcknowledgePacket(packetFromBtoC, successAck.Acknowledgement())
			suite.Require().NoError(err)

			// Ack back to A
			err = suite.pathAtoB.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			err = suite.pathAtoB.EndpointA.AcknowledgePacket(packetFromAtoB, successAck.Acknowledgement())
			suite.Require().NoError(err)

			suite.Require().Empty(GetSimApp(suite.chainA).MockContractKeeper.Counters, "chainA's callbacks counter map is not empty")

			chainBCallbackMap := GetSimApp(suite.chainB).MockContractKeeper.Counters
			suite.Require().Equal(tc.expCallbackMapOnChainB, chainBCallbackMap, "chainC: expected callback counters map to be %v, got %v instead", tc.expCallbackMapOnChainB, chainBCallbackMap)

			chainCCallbackMap := GetSimApp(suite.chainC).MockContractKeeper.Counters
			suite.Require().Equal(tc.expCallbackMapOnChainC, chainCCallbackMap, "chainC: expected callback counters map to be %v, got %v instead", tc.expCallbackMapOnChainC, chainCCallbackMap)
		})
	}
}
