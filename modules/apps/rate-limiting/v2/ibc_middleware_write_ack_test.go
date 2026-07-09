package v2_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/keeper"
	ratelimitingtypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	ratelimitingv2 "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/v2"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
	ibcmockv2 "github.com/cosmos/ibc-go/v11/testing/mock/v2"
)

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
