package v2_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/keeper"
	ratelimitingtypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	ratelimitingv2 "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/v2"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	channelkeeperv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/keeper"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v11/modules/core/api"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
	ibcmockv2 "github.com/cosmos/ibc-go/v11/testing/mock/v2"
)

const rateLimitChannelValue = int64(1000)

type mockPacketUnmarshalerModule struct {
	ibcmockv2.IBCModule

	called     bool
	payload    channeltypesv2.Payload
	packetData any
}

func (m *mockPacketUnmarshalerModule) UnmarshalPacketData(payload channeltypesv2.Payload) (any, error) {
	m.called = true
	m.payload = payload
	return m.packetData, nil
}

type mockWriteAckWrapper struct {
	called  bool
	ack     channeltypesv2.Acknowledgement
	client  string
	seq     uint64
	callErr error
}

func (m *mockWriteAckWrapper) WriteAcknowledgement(_ sdk.Context, clientID string, sequence uint64, ack channeltypesv2.Acknowledgement) error {
	m.called = true
	m.client = clientID
	m.seq = sequence
	m.ack = ack
	return m.callErr
}

type mockChannelKeeperV2 struct {
	packet channeltypesv2.Packet
	found  bool
}

func (m mockChannelKeeperV2) GetAsyncPacket(sdk.Context, string, uint64) (channeltypesv2.Packet, bool) {
	return m.packet, m.found
}

func TestNewIBCMiddleware(t *testing.T) {
	testCases := []struct {
		name          string
		instantiateFn func()
		expPanic      string
	}{
		{
			name: "success",
			instantiateFn: func() {
				_ = ratelimitingv2.NewIBCMiddleware(keeper.Keeper{}, ibcmockv2.IBCModule{}, &channelkeeperv2.Keeper{}, &channelkeeperv2.Keeper{})
			},
		},
		{
			name: "failure: nil write acknowledgement wrapper",
			instantiateFn: func() {
				_ = ratelimitingv2.NewIBCMiddleware(keeper.Keeper{}, ibcmockv2.IBCModule{}, nil, &channelkeeperv2.Keeper{})
			},
			expPanic: "write acknowledgement wrapper cannot be nil",
		},
		{
			name: "failure: nil channel keeper v2",
			instantiateFn: func() {
				_ = ratelimitingv2.NewIBCMiddleware(keeper.Keeper{}, ibcmockv2.IBCModule{}, &channelkeeperv2.Keeper{}, nil)
			},
			expPanic: "channel keeper v2 cannot be nil",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expPanic == "" {
				require.NotPanics(t, tc.instantiateFn)
			} else {
				require.PanicsWithError(t, tc.expPanic, tc.instantiateFn)
			}
		})
	}
}

func TestUnmarshalPacketData(t *testing.T) {
	payload := channeltypesv2.Payload{Value: []byte("payload")}
	expPacketData := "packet data"
	app := &mockPacketUnmarshalerModule{packetData: expPacketData}

	middleware := ratelimitingv2.NewIBCMiddleware(keeper.Keeper{}, app, &channelkeeperv2.Keeper{}, &channelkeeperv2.Keeper{})
	require.Implements(t, (*api.PacketUnmarshalerModuleV2)(nil), middleware)

	packetData, err := middleware.UnmarshalPacketData(payload)
	require.NoError(t, err)
	require.True(t, app.called)
	require.Equal(t, payload, app.payload)
	require.Equal(t, expPacketData, packetData)
}

func TestV2ToV1Packet(t *testing.T) {
	const (
		sourceClient      = "sourceClient"
		destinationClient = "destinationClient"
		sequence          = uint64(1)
	)

	payloadValue := transfertypes.FungibleTokenPacketData{
		Denom:    "denom",
		Amount:   "100",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     "memo",
	}

	mustMarshalPacketData := func(encoding string) []byte {
		bz, err := transfertypes.MarshalPacketData(payloadValue, transfertypes.V1, encoding)
		require.NoError(t, err)
		return bz
	}

	testCases := []struct {
		name     string
		encoding string
		value    []byte
		expErr   bool
	}{
		{
			name:     "success: JSON encoding",
			encoding: transfertypes.EncodingJSON,
			value:    mustMarshalPacketData(transfertypes.EncodingJSON),
		},
		{
			name:     "success: ABI encoding",
			encoding: transfertypes.EncodingABI,
			value:    mustMarshalPacketData(transfertypes.EncodingABI),
		},
		{
			name:     "success: protobuf encoding",
			encoding: transfertypes.EncodingProtobuf,
			value:    mustMarshalPacketData(transfertypes.EncodingProtobuf),
		},
		{
			name:     "failure: nil payload",
			encoding: transfertypes.EncodingABI,
			expErr:   true,
		},
		{
			name:     "failure: empty payload",
			encoding: transfertypes.EncodingABI,
			value:    []byte{},
			expErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := channeltypesv2.Payload{
				SourcePort:      "sourcePort",
				DestinationPort: "destinationPort",
				Version:         transfertypes.V1,
				Encoding:        tc.encoding,
				Value:           tc.value,
			}

			v1Packet, err := ratelimitingv2.V2ToV1Packet(payload, sourceClient, destinationClient, sequence)
			if tc.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, sequence, v1Packet.Sequence)
			require.Equal(t, payload.SourcePort, v1Packet.SourcePort)
			require.Equal(t, sourceClient, v1Packet.SourceChannel)
			require.Equal(t, payload.DestinationPort, v1Packet.DestinationPort)
			require.Equal(t, destinationClient, v1Packet.DestinationChannel)

			var v1PacketData transfertypes.FungibleTokenPacketData
			err = json.Unmarshal(v1Packet.Data, &v1PacketData)
			require.NoError(t, err)
			require.Equal(t, payloadValue, v1PacketData)
		})
	}
}

func TestWriteAcknowledgement(t *testing.T) {
	const (
		sequence          = uint64(1)
		sourceClient      = "sourceClient"
		destinationClient = "destinationClient"
		uosmo             = "uosmo"
		transferPort      = "transfer"
	)

	packetAmount := sdkmath.NewInt(10)
	errorAck := channeltypesv2.NewAcknowledgement(channeltypesv2.ErrorAcknowledgement[:])
	successAck := channeltypesv2.NewAcknowledgement([]byte("success"))
	writeAckErr := "write acknowledgement failed"

	testCases := []struct {
		name              string
		ack               channeltypesv2.Acknowledgement
		asyncFound        bool
		malleatePayload   func(*channeltypesv2.Payload)
		writeAckErr       error
		expErrContains    string
		expWriteAckCalled bool
		checkInflow       bool
		expectedInflow    sdkmath.Int
	}{
		{
			name:              "success: error acknowledgement undoes receive inflow",
			ack:               errorAck,
			asyncFound:        true,
			expWriteAckCalled: true,
			checkInflow:       true,
			expectedInflow:    sdkmath.NewInt(90),
		},
		{
			name:              "success: success acknowledgement does not undo receive inflow",
			ack:               successAck,
			asyncFound:        true,
			expWriteAckCalled: true,
			checkInflow:       true,
			expectedInflow:    sdkmath.NewInt(100),
		},
		{
			name:           "failure: missing async packet",
			ack:            errorAck,
			checkInflow:    true,
			expectedInflow: sdkmath.NewInt(100),
			expErrContains: "async packet not found",
		},
		{
			name:       "failure: async packet cannot be converted",
			ack:        errorAck,
			asyncFound: true,
			malleatePayload: func(payload *channeltypesv2.Payload) {
				payload.Encoding = "invalid"
				payload.Value = []byte("invalid packet data")
			},
			expErrContains: "invalid encoding",
		},
		{
			name:              "failure: write acknowledgement wrapper returns error",
			ack:               successAck,
			asyncFound:        true,
			writeAckErr:       errors.New(writeAckErr),
			expErrContains:    writeAckErr,
			expWriteAckCalled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			coordinator := ibctesting.NewCoordinator(t, 1)
			chain := coordinator.GetChain(ibctesting.GetChainID(1))
			ctx := chain.GetContext()

			packetData := transfertypes.FungibleTokenPacketData{
				Denom:    uosmo,
				Amount:   packetAmount.String(),
				Sender:   "sender",
				Receiver: "receiver",
			}
			packetDataBz, err := transfertypes.MarshalPacketData(packetData, transfertypes.V1, transfertypes.EncodingJSON)
			require.NoError(t, err)

			payload := channeltypesv2.Payload{
				SourcePort:      transferPort,
				DestinationPort: transferPort,
				Version:         transfertypes.V1,
				Encoding:        transfertypes.EncodingJSON,
				Value:           packetDataBz,
			}
			if tc.malleatePayload != nil {
				tc.malleatePayload(&payload)
			}

			packet := channeltypesv2.NewPacket(sequence, sourceClient, destinationClient, 0, payload)
			var packetInfo keeper.RateLimitedPacketInfo
			if tc.checkInflow {
				packetInfo, err = recvPacketInfo(payload, sourceClient, destinationClient, sequence)
				require.NoError(t, err)

				chain.GetSimApp().RateLimitKeeper.SetRateLimit(ctx, ratelimitingtypes.RateLimit{
					Path: &ratelimitingtypes.Path{Denom: packetInfo.Denom, ChannelOrClientId: packetInfo.ChannelID},
					Flow: &ratelimitingtypes.Flow{Inflow: sdkmath.NewInt(100)},
				})
				if tc.asyncFound {
					err = chain.GetSimApp().RateLimitKeeper.SetPendingReceivePacket(ctx, packetInfo.ChannelID, sequence, packetInfo.Denom)
					require.NoError(t, err)
				}
			}

			writeAckWrapper := &mockWriteAckWrapper{callErr: tc.writeAckErr}
			mw := ratelimitingv2.NewIBCMiddleware(
				*chain.GetSimApp().RateLimitKeeper,
				ibcmockv2.IBCModule{},
				writeAckWrapper,
				mockChannelKeeperV2{packet: packet, found: tc.asyncFound},
			)

			err = mw.WriteAcknowledgement(ctx, destinationClient, sequence, tc.ack)
			if tc.expErrContains != "" {
				require.ErrorContains(t, err, tc.expErrContains)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expWriteAckCalled, writeAckWrapper.called)
			if tc.expWriteAckCalled {
				require.Equal(t, destinationClient, writeAckWrapper.client)
				require.Equal(t, sequence, writeAckWrapper.seq)
				require.Equal(t, tc.ack, writeAckWrapper.ack)
			}

			if tc.checkInflow {
				rateLimit, found := chain.GetSimApp().RateLimitKeeper.GetRateLimit(ctx, packetInfo.Denom, packetInfo.ChannelID)
				require.True(t, found)
				require.Equal(t, tc.expectedInflow, rateLimit.Flow.Inflow)

				found, err = chain.GetSimApp().RateLimitKeeper.CheckPacketReceivedDuringCurrentQuota(ctx, packetInfo.ChannelID, sequence, packetInfo.Denom)
				require.NoError(t, err)
				require.False(t, found)
			}
		})
	}
}

func recvPacketInfo(payload channeltypesv2.Payload, sourceClient, destinationClient string, sequence uint64) (keeper.RateLimitedPacketInfo, error) {
	packet, err := ratelimitingv2.V2ToV1Packet(payload, sourceClient, destinationClient, sequence)
	if err != nil {
		return keeper.RateLimitedPacketInfo{}, err
	}

	return keeper.ParsePacketInfo(packet, ratelimitingtypes.PACKET_RECV)
}

type RateLimitMiddlewareTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
	path        *ibctesting.Path
}

func TestRateLimitMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(RateLimitMiddlewareTestSuite))
}

func (s *RateLimitMiddlewareTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.path = ibctesting.NewPath(s.chainA, s.chainB)
	s.path.SetupV2()
}

func (s *RateLimitMiddlewareTestSuite) TestV2TransferSuccessUpdatesFlows() {
	amount := sdkmath.NewInt(10)
	recvDenom := s.voucherDenom()

	s.setRateLimit(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, 100, 100)
	s.setRateLimit(s.chainB, recvDenom, s.path.EndpointB.ClientID, 100, 100)

	senderInitialBalance := s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	payload := s.transferPayload(amount)

	packet, err := s.path.EndpointA.MsgSendPacket(s.timeoutTimestamp(time.Hour), payload)
	s.Require().NoError(err)

	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), amount)
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, true)

	ack, err := s.path.EndpointB.MsgRecvPacketWithAck(packet)
	s.Require().NoError(err)
	s.Require().Len(ack.AppAcknowledgements, 1)
	s.Require().Equal(channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement(), ack.AppAcknowledgements[0])

	s.assertFlow(s.chainB, recvDenom, s.path.EndpointB.ClientID, amount, sdkmath.ZeroInt())
	s.Require().Equal(amount, s.balance(s.chainB, s.chainB.SenderAccount.GetAddress(), recvDenom).Amount)

	err = s.path.EndpointA.MsgAcknowledgePacket(packet, ack)
	s.Require().NoError(err)

	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), amount)
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, false)
	s.Require().Equal(senderInitialBalance.Amount.Sub(amount), s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom).Amount)
}

func (s *RateLimitMiddlewareTestSuite) TestV2TransferSendDenied() {
	amount := sdkmath.NewInt(11)
	s.setRateLimit(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, 1, 100)

	senderInitialBalance := s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	payload := s.transferPayload(amount)

	_, err := s.path.EndpointA.MsgSendPacket(s.timeoutTimestamp(time.Hour), payload)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), ratelimitingtypes.ErrQuotaExceeded.Error())

	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), sdkmath.ZeroInt())
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, 1, false)
	s.Require().Equal(senderInitialBalance, s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom))
}

func (s *RateLimitMiddlewareTestSuite) TestV2TransferReceiveDeniedUndoSendOnErrorAck() {
	amount := sdkmath.NewInt(11)
	recvDenom := s.voucherDenom()

	s.setRateLimit(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, 100, 100)
	s.setRateLimit(s.chainB, recvDenom, s.path.EndpointB.ClientID, 100, 1)

	senderInitialBalance := s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	payload := s.transferPayload(amount)

	packet, err := s.path.EndpointA.MsgSendPacket(s.timeoutTimestamp(time.Hour), payload)
	s.Require().NoError(err)
	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), amount)
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, true)

	ack, err := s.path.EndpointB.MsgRecvPacketWithAck(packet)
	s.Require().NoError(err)
	s.Require().Len(ack.AppAcknowledgements, 1)
	s.Require().Equal(channeltypesv2.ErrorAcknowledgement[:], ack.AppAcknowledgements[0])
	s.assertFlow(s.chainB, recvDenom, s.path.EndpointB.ClientID, sdkmath.ZeroInt(), sdkmath.ZeroInt())
	s.Require().True(s.balance(s.chainB, s.chainB.SenderAccount.GetAddress(), recvDenom).IsZero())

	err = s.path.EndpointA.MsgAcknowledgePacket(packet, ack)
	s.Require().NoError(err)

	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), sdkmath.ZeroInt())
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, false)
	s.Require().Equal(senderInitialBalance, s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom))
}

func (s *RateLimitMiddlewareTestSuite) TestV2TransferTimeoutUndoSend() {
	amount := sdkmath.NewInt(10)
	s.setRateLimit(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, 100, 100)

	senderInitialBalance := s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	payload := s.transferPayload(amount)

	packet, err := s.path.EndpointA.MsgSendPacket(s.timeoutTimestamp(time.Second), payload)
	s.Require().NoError(err)
	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), amount)
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, true)
	s.Require().Equal(senderInitialBalance.Amount.Sub(amount), s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom).Amount)

	s.Require().NoError(s.path.EndpointA.UpdateClient())
	err = s.path.EndpointA.MsgTimeoutPacket(packet)
	s.Require().NoError(err)

	s.assertFlow(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, sdkmath.ZeroInt(), sdkmath.ZeroInt())
	s.assertPendingPacket(s.chainA, sdk.DefaultBondDenom, s.path.EndpointA.ClientID, packet.Sequence, false)
	s.Require().Equal(senderInitialBalance, s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom))
}

func (s *RateLimitMiddlewareTestSuite) setRateLimit(chain *ibctesting.TestChain, denom, clientID string, maxPercentSend, maxPercentRecv int64) {
	chain.GetSimApp().RateLimitKeeper.SetRateLimit(chain.GetContext(), ratelimitingtypes.RateLimit{
		Path: &ratelimitingtypes.Path{
			Denom:             denom,
			ChannelOrClientId: clientID,
		},
		Quota: &ratelimitingtypes.Quota{
			MaxPercentSend: sdkmath.NewInt(maxPercentSend),
			MaxPercentRecv: sdkmath.NewInt(maxPercentRecv),
			DurationHours:  1,
		},
		Flow: &ratelimitingtypes.Flow{
			Inflow:       sdkmath.ZeroInt(),
			Outflow:      sdkmath.ZeroInt(),
			ChannelValue: sdkmath.NewInt(rateLimitChannelValue),
		},
	})
}

func (s *RateLimitMiddlewareTestSuite) transferPayload(amount sdkmath.Int) channeltypesv2.Payload {
	packetData := transfertypes.NewFungibleTokenPacketData(
		sdk.DefaultBondDenom,
		amount.String(),
		s.chainA.SenderAccount.GetAddress().String(),
		s.chainB.SenderAccount.GetAddress().String(),
		"",
	)
	bz := s.chainA.Codec.MustMarshal(&packetData)

	return channeltypesv2.NewPayload(transfertypes.PortID, transfertypes.PortID, transfertypes.V1, transfertypes.EncodingProtobuf, bz)
}

func (s *RateLimitMiddlewareTestSuite) voucherDenom() string {
	return transfertypes.NewDenom(
		sdk.DefaultBondDenom,
		transfertypes.NewHop(transfertypes.PortID, s.path.EndpointB.ClientID),
	).IBCDenom()
}

func (s *RateLimitMiddlewareTestSuite) timeoutTimestamp(duration time.Duration) uint64 {
	return uint64(s.chainB.GetContext().BlockTime().Add(duration).Unix())
}

func (s *RateLimitMiddlewareTestSuite) assertFlow(chain *ibctesting.TestChain, denom, clientID string, expectedInflow, expectedOutflow sdkmath.Int) {
	rateLimit, found := chain.GetSimApp().RateLimitKeeper.GetRateLimit(chain.GetContext(), denom, clientID)
	s.Require().True(found)
	s.Require().True(rateLimit.Flow.Inflow.Equal(expectedInflow), "expected inflow %s, got %s", expectedInflow, rateLimit.Flow.Inflow)
	s.Require().True(rateLimit.Flow.Outflow.Equal(expectedOutflow), "expected outflow %s, got %s", expectedOutflow, rateLimit.Flow.Outflow)
}

func (s *RateLimitMiddlewareTestSuite) assertPendingPacket(chain *ibctesting.TestChain, denom, clientID string, sequence uint64, expected bool) {
	found, err := chain.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(chain.GetContext(), clientID, sequence, denom)
	s.Require().NoError(err)
	s.Require().Equal(expected, found)
}

func (s *RateLimitMiddlewareTestSuite) balance(chain *ibctesting.TestChain, address sdk.AccAddress, denom string) sdk.Coin {
	return chain.GetSimApp().BankKeeper.GetBalance(chain.GetContext(), address, denom)
}
