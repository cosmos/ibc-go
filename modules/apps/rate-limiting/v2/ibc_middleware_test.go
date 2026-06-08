package v2 // nolint

import (
	"encoding/json"
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

type mockIBCModule struct{}

func (mockIBCModule) OnSendPacket(sdk.Context, string, string, uint64, channeltypesv2.Payload, sdk.AccAddress) error {
	return nil
}

func (mockIBCModule) OnRecvPacket(sdk.Context, string, string, uint64, channeltypesv2.Payload, sdk.AccAddress) channeltypesv2.RecvPacketResult {
	return channeltypesv2.RecvPacketResult{}
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

func (m mockChannelKeeperV2) GetAsyncPacket(sdk.Context, string, uint64) (channeltypesv2.Packet, bool) {
	return m.packet, m.found
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

	testCases := []struct {
		name           string
		ack            channeltypesv2.Acknowledgement
		asyncFound     bool
		expectedInflow sdkmath.Int
	}{
		{
			name:           "success: error acknowledgement undoes receive inflow",
			ack:            errorAck,
			asyncFound:     true,
			expectedInflow: sdkmath.NewInt(90),
		},
		{
			name:           "success: success acknowledgement does not undo receive inflow",
			ack:            successAck,
			asyncFound:     true,
			expectedInflow: sdkmath.NewInt(100),
		},
		{
			name:           "success: missing async packet does not undo receive inflow",
			ack:            errorAck,
			asyncFound:     false,
			expectedInflow: sdkmath.NewInt(100),
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
			packet := channeltypesv2.NewPacket(sequence, sourceClient, destinationClient, 0, payload)
			v1Packet, err := v2ToV1Packet(payload, sourceClient, destinationClient, sequence)
			require.NoError(t, err)
			packetInfo, err := keeper.ParsePacketInfo(v1Packet, ratelimitingtypes.PACKET_RECV)
			require.NoError(t, err)

			chain.GetSimApp().RateLimitKeeper.SetRateLimit(ctx, ratelimitingtypes.RateLimit{
				Path: &ratelimitingtypes.Path{Denom: packetInfo.Denom, ChannelOrClientId: packetInfo.ChannelID},
				Flow: &ratelimitingtypes.Flow{Inflow: sdkmath.NewInt(100)},
			})

			writeAckWrapper := &mockWriteAckWrapper{}
			mw := NewIBCMiddleware(
				*chain.GetSimApp().RateLimitKeeper,
				mockIBCModule{},
				writeAckWrapper,
				mockChannelKeeperV2{packet: packet, found: tc.asyncFound},
			)

			err = mw.WriteAcknowledgement(ctx, destinationClient, sequence, tc.ack)
			require.NoError(t, err)
			require.True(t, writeAckWrapper.called)
			require.Equal(t, destinationClient, writeAckWrapper.client)
			require.Equal(t, sequence, writeAckWrapper.seq)
			require.Equal(t, tc.ack, writeAckWrapper.ack)

			rateLimit, found := chain.GetSimApp().RateLimitKeeper.GetRateLimit(ctx, packetInfo.Denom, packetInfo.ChannelID)
			require.True(t, found)
			require.Equal(t, tc.expectedInflow, rateLimit.Flow.Inflow)
		})
	}
}

func TestV2ToV1Packet_WithJSONEncoding(t *testing.T) {
	payloadValue := transfertypes.FungibleTokenPacketData{
		Denom:    "denom",
		Amount:   "100",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     "memo",
	}
	payloadValueBz, err := transfertypes.MarshalPacketData(payloadValue, transfertypes.V1, transfertypes.EncodingJSON)
	require.NoError(t, err)

	payload := channeltypesv2.Payload{
		SourcePort:      "sourcePort",
		DestinationPort: "destinationPort",
		Version:         transfertypes.V1,
		Encoding:        transfertypes.EncodingJSON,
		Value:           payloadValueBz,
	}

	v1Packet, err := v2ToV1Packet(payload, "sourceClient", "destinationClient", 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), v1Packet.Sequence)
	require.Equal(t, payload.SourcePort, v1Packet.SourcePort)
	require.Equal(t, "sourceClient", v1Packet.SourceChannel)
	require.Equal(t, payload.DestinationPort, v1Packet.DestinationPort)
	require.Equal(t, "destinationClient", v1Packet.DestinationChannel)

	var v1PacketData transfertypes.FungibleTokenPacketData
	err = json.Unmarshal(v1Packet.Data, &v1PacketData)
	require.NoError(t, err)
	require.Equal(t, payloadValue, v1PacketData)
}

func TestV2ToV1Packet_WithABIEncoding(t *testing.T) {
	payloadValue := transfertypes.FungibleTokenPacketData{
		Denom:    "denom",
		Amount:   "100",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     "memo",
	}

	payloadValueBz, err := transfertypes.MarshalPacketData(payloadValue, transfertypes.V1, transfertypes.EncodingABI)
	require.NoError(t, err)

	payload := channeltypesv2.Payload{
		SourcePort:      "sourcePort",
		DestinationPort: "destinationPort",
		Version:         transfertypes.V1,
		Encoding:        transfertypes.EncodingABI,
		Value:           payloadValueBz,
	}

	v1Packet, err := v2ToV1Packet(payload, "sourceClient", "destinationClient", 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), v1Packet.Sequence)
	require.Equal(t, payload.SourcePort, v1Packet.SourcePort)
	require.Equal(t, "sourceClient", v1Packet.SourceChannel)
	require.Equal(t, payload.DestinationPort, v1Packet.DestinationPort)
	require.Equal(t, "destinationClient", v1Packet.DestinationChannel)

	var v1PacketData transfertypes.FungibleTokenPacketData
	err = json.Unmarshal(v1Packet.Data, &v1PacketData)
	require.NoError(t, err)
	require.Equal(t, payloadValue, v1PacketData)
}

func TestV2ToV1Packet_WithProtobufEncoding(t *testing.T) {
	payloadValue := transfertypes.FungibleTokenPacketData{
		Denom:    "denom",
		Amount:   "100",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     "memo",
	}

	payloadValueBz, err := transfertypes.MarshalPacketData(payloadValue, transfertypes.V1, transfertypes.EncodingProtobuf)
	require.NoError(t, err)

	payload := channeltypesv2.Payload{
		SourcePort:      "sourcePort",
		DestinationPort: "destinationPort",
		Version:         transfertypes.V1,
		Encoding:        transfertypes.EncodingProtobuf,
		Value:           payloadValueBz,
	}

	v1Packet, err := v2ToV1Packet(payload, "sourceClient", "destinationClient", 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), v1Packet.Sequence)
	require.Equal(t, payload.SourcePort, v1Packet.SourcePort)
	require.Equal(t, "sourceClient", v1Packet.SourceChannel)
	require.Equal(t, payload.DestinationPort, v1Packet.DestinationPort)
	require.Equal(t, "destinationClient", v1Packet.DestinationChannel)

	var v1PacketData transfertypes.FungibleTokenPacketData
	err = json.Unmarshal(v1Packet.Data, &v1PacketData)
	require.NoError(t, err)
	require.Equal(t, payloadValue, v1PacketData)
}

func TestV2ToV1Packet_WithNilPayload(t *testing.T) {
	payload := channeltypesv2.Payload{
		SourcePort:      "sourcePort",
		DestinationPort: "destinationPort",
		Version:         transfertypes.V1,
		Encoding:        transfertypes.EncodingABI,
		Value:           nil,
	}

	_, err := v2ToV1Packet(payload, "sourceClient", "destinationClient", 1)
	require.Error(t, err)
}

func TestV2ToV1Packet_WithEmptyPayload(t *testing.T) {
	payload := channeltypesv2.Payload{
		SourcePort:      "sourcePort",
		DestinationPort: "destinationPort",
		Version:         transfertypes.V1,
		Encoding:        transfertypes.EncodingABI,
		Value:           []byte{},
	}

	_, err := v2ToV1Packet(payload, "sourceClient", "destinationClient", 1)
	require.Error(t, err)
}
