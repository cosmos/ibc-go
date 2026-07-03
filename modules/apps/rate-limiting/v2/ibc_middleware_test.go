package v2 // nolint

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/keeper"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
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
	called bool
	ack    channeltypesv2.Acknowledgement
	client string
	seq    uint64
}

func (m *mockWriteAckWrapper) WriteAcknowledgement(_ sdk.Context, clientID string, sequence uint64, ack channeltypesv2.Acknowledgement) error {
	m.called = true
	m.client = clientID
	m.seq = sequence
	m.ack = ack
	return nil
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
