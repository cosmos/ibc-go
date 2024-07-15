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
func (s *CallbacksForwardingTestSuite) TestForwardingWithMemoCallback() {
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
			name:                   "ack and recv callback",
			testMemo:               fmt.Sprintf(`{"src_callback": {"address": "%s"}, "dest_callback": {"address": "%s"}}`, simapp.SuccessContract, simapp.SuccessContract),
			expCallbackMapOnChainB: map[callbacktypes.CallbackType]int{callbacktypes.CallbackTypeAcknowledgementPacket: 1},
			expCallbackMapOnChainC: map[callbacktypes.CallbackType]int{callbacktypes.CallbackTypeReceivePacket: 1},
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
		s.Run(tc.name, func() {
			s.SetupTest()

			coinOnA := ibctesting.TestCoin
			sender := s.chainA.SenderAccounts[0].SenderAccount
			receiver := s.chainC.SenderAccounts[0].SenderAccount
			forwarding := types.NewForwarding(false, types.NewHop(
				s.pathBtoC.EndpointA.ChannelConfig.PortID,
				s.pathBtoC.EndpointA.ChannelID,
			))
			successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

			transferMsg := types.NewMsgTransfer(
				s.pathAtoB.EndpointA.ChannelConfig.PortID,
				s.pathAtoB.EndpointA.ChannelID,
				sdk.NewCoins(coinOnA),
				sender.GetAddress().String(),
				receiver.GetAddress().String(),
				clienttypes.ZeroHeight(),
				s.chainA.GetTimeoutTimestamp(),
				tc.testMemo,
				forwarding,
			)

			result, err := s.chainA.SendMsgs(transferMsg)
			s.Require().NoError(err) // message committed

			// parse the packet from result events and recv packet on chainB
			packetFromAtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
			s.Require().NoError(err)
			s.Require().NotNil(packetFromAtoB)

			err = s.pathAtoB.EndpointB.UpdateClient()
			s.Require().NoError(err)

			result, err = s.pathAtoB.EndpointB.RecvPacketWithResult(packetFromAtoB)
			s.Require().NoError(err)
			s.Require().NotNil(result)

			packetFromBtoC, err := ibctesting.ParsePacketFromEvents(result.Events)
			s.Require().NoError(err)
			s.Require().NotNil(packetFromBtoC)

			err = s.pathBtoC.EndpointA.UpdateClient()
			s.Require().NoError(err)

			err = s.pathBtoC.EndpointB.UpdateClient()
			s.Require().NoError(err)

			result, err = s.pathBtoC.EndpointB.RecvPacketWithResult(packetFromBtoC)
			s.Require().NoError(err)
			s.Require().NotNil(result)

			packetOnC, err := ibctesting.ParseRecvPacketFromEvents(result.Events)
			s.Require().NoError(err)
			s.Require().NotNil(packetOnC)

			// Ack back to B
			err = s.pathBtoC.EndpointB.UpdateClient()
			s.Require().NoError(err)

			err = s.pathBtoC.EndpointA.AcknowledgePacket(packetFromBtoC, successAck.Acknowledgement())
			s.Require().NoError(err)

			// Ack back to A
			err = s.pathAtoB.EndpointA.UpdateClient()
			s.Require().NoError(err)

			err = s.pathAtoB.EndpointA.AcknowledgePacket(packetFromAtoB, successAck.Acknowledgement())
			s.Require().NoError(err)

			// We never expect chainA to have executed any callback
			s.Require().Empty(GetSimApp(s.chainA).MockContractKeeper.Counters, "chainA's callbacks counter map is not empty")

			// We expect chainB to have executed callbacks when the memo is of type `src_callback`
			chainBCallbackMap := GetSimApp(s.chainB).MockContractKeeper.Counters
			s.Require().Equal(tc.expCallbackMapOnChainB, chainBCallbackMap, "chainB: expected callback counters map to be %v, got %v instead", tc.expCallbackMapOnChainB, chainBCallbackMap)

			// We expect chainC to have executed callbacks when the memo is of type `dest_callback`
			chainCCallbackMap := GetSimApp(s.chainC).MockContractKeeper.Counters
			s.Require().Equal(tc.expCallbackMapOnChainC, chainCCallbackMap, "chainC: expected callback counters map to be %v, got %v instead", tc.expCallbackMapOnChainC, chainCCallbackMap)
		})
	}
}
