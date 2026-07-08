package v2 // nolint

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/keeper"
	ratelimitingtypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

type mockIBCModule struct {
	recvResult channeltypesv2.RecvPacketResult
}

func (mockIBCModule) OnSendPacket(sdk.Context, string, string, uint64, channeltypesv2.Payload, sdk.AccAddress) error {
	return nil
}

func (m mockIBCModule) OnRecvPacket(sdk.Context, string, string, uint64, channeltypesv2.Payload, sdk.AccAddress) channeltypesv2.RecvPacketResult {
	return m.recvResult
}

func (mockIBCModule) OnTimeoutPacket(sdk.Context, string, string, uint64, channeltypesv2.Payload, sdk.AccAddress) error {
	return nil
}

func (mockIBCModule) OnAcknowledgementPacket(sdk.Context, string, string, uint64, []byte, channeltypesv2.Payload, sdk.AccAddress) error {
	return nil
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

const (
	testPacketSender   = "sender"
	testPacketReceiver = "receiver"
)

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
				_ = NewIBCMiddleware(keeper.Keeper{}, mockIBCModule{}, &mockWriteAckWrapper{}, mockChannelKeeperV2{})
			},
		},
		{
			name: "failure: nil write acknowledgement wrapper",
			instantiateFn: func() {
				_ = NewIBCMiddleware(keeper.Keeper{}, mockIBCModule{}, nil, mockChannelKeeperV2{})
			},
			expPanic: "write acknowledgement wrapper cannot be nil",
		},
		{
			name: "failure: nil channel keeper v2",
			instantiateFn: func() {
				_ = NewIBCMiddleware(keeper.Keeper{}, mockIBCModule{}, &mockWriteAckWrapper{}, nil)
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
				Sender:   testPacketSender,
				Receiver: testPacketReceiver,
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
				v1Packet, err := v2ToV1Packet(payload, sourceClient, destinationClient, sequence)
				require.NoError(t, err)
				packetInfo, err = keeper.ParsePacketInfo(v1Packet, ratelimitingtypes.PACKET_RECV)
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
			mw := NewIBCMiddleware(
				*chain.GetSimApp().RateLimitKeeper,
				mockIBCModule{},
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

func TestOnRecvPacket(t *testing.T) {
	const (
		sequence          = uint64(1)
		sourceClient      = "sourceClient"
		destinationClient = "destinationClient"
		uosmo             = "uosmo"
		transferPort      = "transfer"
	)

	packetAmount := sdkmath.NewInt(10)

	testCases := []struct {
		name         string
		resultStatus channeltypesv2.PacketStatus
		expPending   bool
	}{
		{
			name:         "success: sync result removes pending receive packet",
			resultStatus: channeltypesv2.PacketStatus_Success,
		},
		{
			name:         "success: async result leaves pending receive packet",
			resultStatus: channeltypesv2.PacketStatus_Async,
			expPending:   true,
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
				Sender:   testPacketSender,
				Receiver: testPacketReceiver,
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
			v1Packet, err := v2ToV1Packet(payload, sourceClient, destinationClient, sequence)
			require.NoError(t, err)
			packetInfo, err := keeper.ParsePacketInfo(v1Packet, ratelimitingtypes.PACKET_RECV)
			require.NoError(t, err)

			chain.GetSimApp().RateLimitKeeper.SetRateLimit(ctx, ratelimitingtypes.RateLimit{
				Path: &ratelimitingtypes.Path{Denom: packetInfo.Denom, ChannelOrClientId: packetInfo.ChannelID},
				Quota: &ratelimitingtypes.Quota{
					MaxPercentSend: sdkmath.NewInt(100),
					MaxPercentRecv: sdkmath.NewInt(100),
				},
				Flow: &ratelimitingtypes.Flow{
					Inflow:       sdkmath.ZeroInt(),
					Outflow:      sdkmath.ZeroInt(),
					ChannelValue: sdkmath.NewInt(100),
				},
			})

			mw := NewIBCMiddleware(
				*chain.GetSimApp().RateLimitKeeper,
				mockIBCModule{recvResult: channeltypesv2.RecvPacketResult{Status: tc.resultStatus}},
				&mockWriteAckWrapper{},
				mockChannelKeeperV2{},
			)

			result := mw.OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, nil)
			require.Equal(t, tc.resultStatus, result.Status)

			found, err := chain.GetSimApp().RateLimitKeeper.CheckPacketReceivedDuringCurrentQuota(ctx, packetInfo.ChannelID, sequence, packetInfo.Denom)
			require.NoError(t, err)
			require.Equal(t, tc.expPending, found)
		})
	}
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

			v1Packet, err := v2ToV1Packet(payload, sourceClient, destinationClient, sequence)
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
