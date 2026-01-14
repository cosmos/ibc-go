package ibccallbacks_test

import (
	"fmt"
	"time"

	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	gmp "github.com/cosmos/ibc-go/v10/modules/apps/27-gmp"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/testing/simapp"
	callbacktypes "github.com/cosmos/ibc-go/v10/modules/apps/callbacks/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *CallbacksTestSuite) TestGMPCallbacks() {
	// GMP auto-registers source callbacks - sender is always the callback address
	// This means even with no memo, source callbacks are executed
	testCases := []struct {
		name        string
		gmpMemo     string
		expCallback callbacktypes.CallbackType
		expSuccess  bool
	}{
		{
			"success: gmp tx with no memo - auto source callback",
			"",
			callbacktypes.CallbackTypeAcknowledgementPacket,
			true,
		},
		{
			"success: src_callback in memo is ignored - sender used instead",
			fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
			callbacktypes.CallbackTypeAcknowledgementPacket,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupGMPTest()

			s.ExecuteGMP(tc.gmpMemo)
			s.AssertHasExecutedExpectedCallback(tc.expCallback, tc.expSuccess)
		})
	}
}

func (s *CallbacksTestSuite) TestGMPDestCallbacks() {
	// Test dest callbacks - note that GMP also auto-registers source callbacks,
	// so both source and dest callbacks will fire
	testCases := []struct {
		name       string
		gmpMemo    string
		expSuccess bool
	}{
		{
			"success: dest callback",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.SuccessContract),
			true,
		},
		{
			"success: dest callback with other json fields",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}, "something_else": {}}`, simapp.SuccessContract),
			true,
		},
		{
			"failure: dest callback with low gas (panic)",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.OogPanicContract),
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupGMPTest()

			s.ExecuteGMP(tc.gmpMemo)

			sourceStatefulCounter := GetSimApp(s.chainA).MockContractKeeper.GetStateEntryCounter(s.chainA.GetContext())
			destStatefulCounter := GetSimApp(s.chainB).MockContractKeeper.GetStateEntryCounter(s.chainB.GetContext())

			// SendPacket + Acknowledgement = 2 entries
			s.Require().Equal(uint8(2), sourceStatefulCounter, "source callbacks should fire")

			if tc.expSuccess {
				s.Require().Equal(uint8(1), destStatefulCounter, "dest callback should fire on success")
			} else {
				s.Require().Equal(uint8(0), destStatefulCounter, "dest callback should not fire on failure")
			}
		})
	}
}

func (s *CallbacksTestSuite) TestGMPUnmarshalPacketData() {
	s.SetupGMPTest()

	memo := fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.SuccessContract)
	packetData := types.NewGMPPacketData(
		s.chainA.SenderAccount.GetAddress().String(),
		s.chainB.SenderAccount.GetAddress().String(),
		[]byte("salt"),
		[]byte("payload"),
		memo,
	)

	dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingProtobuf)
	s.Require().NoError(err)

	payload := channeltypesv2.NewPayload(types.PortID, types.PortID, types.Version, types.EncodingProtobuf, dataBz)

	// Verify packet data implements PacketDataProvider directly
	provider, ok := any(&packetData).(ibcexported.PacketDataProvider)
	s.Require().True(ok, "*GMPPacketData should implement PacketDataProvider")
	destCallback := provider.GetCustomPacketData("dest_callback")
	s.Require().NotNil(destCallback, "dest_callback should not be nil")

	// Simulate callbacks middleware - unmarshal and check interface
	gmpModule := gmp.NewIBCModule(GetSimApp(s.chainB).GMPKeeper)
	unmarshaled, err := gmpModule.UnmarshalPacketData(payload)
	s.Require().NoError(err, "UnmarshalPacketData should not error")

	// Check unmarshaled data implements PacketDataProvider
	unmarshaledProvider, ok := unmarshaled.(ibcexported.PacketDataProvider)
	s.Require().True(ok, "unmarshaled data should implement PacketDataProvider, got %T", unmarshaled)
	unmarshaledDestCallback := unmarshaledProvider.GetCustomPacketData("dest_callback")
	s.Require().NotNil(unmarshaledDestCallback, "unmarshaled dest_callback should not be nil")

	// Verify GetCallbackData extracts callback info correctly
	cbData, isCbPacket, err := callbacktypes.GetCallbackData(
		unmarshaled, types.Version, types.PortID,
		1000000, 1000000, callbacktypes.DestinationCallbackKey,
	)
	s.Require().True(isCbPacket, "should be a callback packet")
	s.Require().NoError(err, "GetCallbackData should not error")
	s.Require().Equal(simapp.SuccessContract, cbData.CallbackAddress, "callback address should match")
}

func (s *CallbacksTestSuite) TestGMPTimeoutCallbacks() {
	testCases := []struct {
		name        string
		gmpMemo     string
		expCallback callbacktypes.CallbackType
		expSuccess  bool
	}{
		{
			"success: gmp timeout with no memo - auto source callback",
			"",
			callbacktypes.CallbackTypeTimeoutPacket,
			true,
		},
		{
			"success: dest callback - not reached on timeout",
			fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, simapp.SuccessContract),
			callbacktypes.CallbackTypeTimeoutPacket,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupGMPTest()

			s.ExecuteGMPTimeout(tc.gmpMemo)
			s.AssertHasExecutedExpectedCallback(tc.expCallback, tc.expSuccess)
		})
	}
}

func (s *CallbacksTestSuite) ExecuteGMP(memo string) {
	s.ExecuteGMPWithSenderAndEvents(memo, s.chainA.SenderAccount.GetAddress().String())
}

func (s *CallbacksTestSuite) ExecuteGMPWithSenderAndEvents(memo, sender string) {
	timeoutTimestamp := uint64(s.chainA.GetContext().BlockTime().Add(time.Minute).Unix())
	destClient := s.path.EndpointB.ClientID

	salt := []byte("salt")
	accountID := types.NewAccountIdentifier(destClient, sender, salt)
	gmpAccountAddr, err := types.BuildAddressPredictable(&accountID)
	s.Require().NoError(err)

	s.fundGMPAccount(gmpAccountAddr)

	recipient := s.chainB.SenderAccount.GetAddress()
	txPayload := s.serializeGMPMsgs(s.newGMPMsgSend(gmpAccountAddr, recipient))

	packetData := types.NewGMPPacketData(
		sender,
		s.chainB.SenderAccount.GetAddress().String(),
		salt,
		txPayload,
		memo,
	)

	dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingProtobuf)
	s.Require().NoError(err)

	payload := channeltypesv2.NewPayload(types.PortID, types.PortID, types.Version, types.EncodingProtobuf, dataBz)

	packet, err := s.path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
	s.Require().NoError(err)

	packetKey := hostv2.PacketCommitmentKey(packet.SourceClient, packet.Sequence)
	proof, proofHeight := s.path.EndpointA.QueryProof(packetKey)
	msg := channeltypesv2.NewMsgRecvPacket(packet, proof, proofHeight, s.chainB.SenderAccount.GetAddress().String())
	res, err := s.chainB.SendMsgs(msg)
	s.Require().NoError(err)

	ackBz, err := ibctesting.ParseAckV2FromEvents(res.Events)
	s.Require().NoError(err)
	var ack channeltypesv2.Acknowledgement
	err = proto.Unmarshal(ackBz, &ack)
	s.Require().NoError(err)

	// Commit block to finalize ack before querying proof
	s.coordinator.CommitBlock(s.chainB)
	err = s.path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	err = s.path.EndpointA.MsgAcknowledgePacket(packet, ack)
	s.Require().NoError(err)
}

func (s *CallbacksTestSuite) ExecuteGMPWithSender(memo, sender string) {
	timeoutTimestamp := uint64(s.chainA.GetContext().BlockTime().Add(time.Minute).Unix())
	destClient := s.path.EndpointB.ClientID

	salt := []byte("salt")
	accountID := types.NewAccountIdentifier(destClient, sender, salt)
	gmpAccountAddr, err := types.BuildAddressPredictable(&accountID)
	s.Require().NoError(err)

	s.fundGMPAccount(gmpAccountAddr)

	recipient := s.chainB.SenderAccount.GetAddress()
	txPayload := s.serializeGMPMsgs(s.newGMPMsgSend(gmpAccountAddr, recipient))

	packetData := types.NewGMPPacketData(
		sender,
		s.chainB.SenderAccount.GetAddress().String(),
		salt,
		txPayload,
		memo,
	)

	dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingProtobuf)
	s.Require().NoError(err)

	payload := channeltypesv2.NewPayload(types.PortID, types.PortID, types.Version, types.EncodingProtobuf, dataBz)

	packet, err := s.path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
	s.Require().NoError(err)

	err = s.path.EndpointA.RelayPacket(packet)
	s.Require().NoError(err)
}

func (s *CallbacksTestSuite) ExecuteGMPTimeout(memo string) {
	sender := s.chainA.SenderAccount.GetAddress().String()
	destClient := s.path.EndpointB.ClientID

	salt := []byte("salt-timeout")
	accountID := types.NewAccountIdentifier(destClient, sender, salt)
	gmpAccountAddr, err := types.BuildAddressPredictable(&accountID)
	s.Require().NoError(err)

	recipient := s.chainB.SenderAccount.GetAddress()
	txPayload := s.serializeGMPMsgs(s.newGMPMsgSend(gmpAccountAddr, recipient))

	packetData := types.NewGMPPacketData(
		sender,
		s.chainB.SenderAccount.GetAddress().String(),
		salt,
		txPayload,
		memo,
	)

	dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingProtobuf)
	s.Require().NoError(err)

	payload := channeltypesv2.NewPayload(types.PortID, types.PortID, types.Version, types.EncodingProtobuf, dataBz)

	// Timeout 1 second in future so SendPacket succeeds, then advance time
	timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Second).Unix())

	packet, err := s.path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
	s.Require().NoError(err)

	// Advance past timeout
	s.coordinator.CommitBlock(s.chainA)
	s.Require().NoError(s.path.EndpointB.UpdateClient())
	s.Require().NoError(s.path.EndpointA.UpdateClient())

	err = s.path.EndpointA.MsgTimeoutPacket(packet)
	s.Require().NoError(err)
}

func (s *CallbacksTestSuite) fundGMPAccount(addr sdk.AccAddress) {
	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000)))
	err := GetSimApp(s.chainB).BankKeeper.SendCoins(
		s.chainB.GetContext(),
		s.chainB.SenderAccount.GetAddress(),
		addr,
		coins,
	)
	s.Require().NoError(err)
}

func (s *CallbacksTestSuite) newGMPMsgSend(from, to sdk.AccAddress) *banktypes.MsgSend {
	return &banktypes.MsgSend{
		FromAddress: from.String(),
		ToAddress:   to.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1000))),
	}
}

func (s *CallbacksTestSuite) serializeGMPMsgs(msgs ...proto.Message) []byte {
	payload, err := types.SerializeCosmosTx(GetSimApp(s.chainB).AppCodec(), msgs)
	s.Require().NoError(err)
	return payload
}
