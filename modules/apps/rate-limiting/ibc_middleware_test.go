package ratelimiting_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ratelimiting "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting"
	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/keeper"
	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v11/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

const (
	testTransferPort = "transfer"
	testSequence     = uint64(1)
	invalidPacketBz  = "invalid packet data"
)

type mockPacketUnmarshalerModule struct {
	recvAck ibcexported.Acknowledgement
}

func (mockPacketUnmarshalerModule) OnChanOpenInit(sdk.Context, channeltypes.Order, []string, string, string, channeltypes.Counterparty, string) (string, error) {
	return "", nil
}

func (mockPacketUnmarshalerModule) OnChanOpenTry(sdk.Context, channeltypes.Order, []string, string, string, channeltypes.Counterparty, string) (string, error) {
	return "", nil
}

func (mockPacketUnmarshalerModule) OnChanOpenAck(sdk.Context, string, string, string, string) error {
	return nil
}

func (mockPacketUnmarshalerModule) OnChanOpenConfirm(sdk.Context, string, string) error {
	return nil
}

func (mockPacketUnmarshalerModule) OnChanCloseInit(sdk.Context, string, string) error {
	return nil
}

func (mockPacketUnmarshalerModule) OnChanCloseConfirm(sdk.Context, string, string) error {
	return nil
}

func (m mockPacketUnmarshalerModule) OnRecvPacket(sdk.Context, string, channeltypes.Packet, sdk.AccAddress) ibcexported.Acknowledgement {
	return m.recvAck
}

func (mockPacketUnmarshalerModule) OnAcknowledgementPacket(sdk.Context, string, channeltypes.Packet, []byte, sdk.AccAddress) error {
	return nil
}

func (mockPacketUnmarshalerModule) OnTimeoutPacket(sdk.Context, string, channeltypes.Packet, sdk.AccAddress) error {
	return nil
}

func (mockPacketUnmarshalerModule) SetICS4Wrapper(porttypes.ICS4Wrapper) {}

func (mockPacketUnmarshalerModule) UnmarshalPacketData(sdk.Context, string, string, []byte) (any, string, error) {
	return nil, "", nil
}

type mockICS4Wrapper struct {
	writeAckCalled bool
}

func (*mockICS4Wrapper) SendPacket(sdk.Context, string, string, clienttypes.Height, uint64, []byte) (uint64, error) {
	return 0, nil
}

func (m *mockICS4Wrapper) WriteAcknowledgement(sdk.Context, ibcexported.PacketI, ibcexported.Acknowledgement) error {
	m.writeAckCalled = true
	return nil
}

func (*mockICS4Wrapper) GetAppVersion(sdk.Context, string, string) (string, bool) {
	return "", false
}

func TestWriteAcknowledgement_NilAck(t *testing.T) {
	middleware := ratelimiting.NewIBCMiddleware(nil)
	packet := channeltypes.Packet{
		Sequence:           1,
		DestinationChannel: "channel-0",
	}

	var ack ibcexported.Acknowledgement
	err := middleware.WriteAcknowledgement(sdk.Context{}, packet, ack)

	require.ErrorIs(t, err, types.ErrAsyncAckNil)
	require.ErrorContains(t, err, "cannot write nil ack for packet channel-0/1")
}

func TestOnRecvPacketRemovesPendingReceivePacketForSyncAck(t *testing.T) {
	coordinator := ibctesting.NewCoordinator(t, 1)
	chain := coordinator.GetChain(ibctesting.GetChainID(1))
	ctx := chain.GetContext()
	ratelimitKeeper := chain.GetSimApp().RateLimitKeeper

	packet, packetInfo := createRecvPacket(t)
	setReceiveRateLimit(ctx, ratelimitKeeper, packetInfo)

	middleware := ratelimiting.NewIBCMiddleware(ratelimitKeeper)
	expectedAck := channeltypes.NewResultAcknowledgement([]byte{1})
	middleware.SetUnderlyingApplication(mockPacketUnmarshalerModule{recvAck: expectedAck})

	ack := middleware.OnRecvPacket(ctx, "", packet, nil)
	require.Equal(t, expectedAck, ack)

	found, err := ratelimitKeeper.CheckPacketReceivedDuringCurrentQuota(ctx, packetInfo.ChannelID, packet.Sequence, packetInfo.Denom)
	require.NoError(t, err)
	require.False(t, found)
}

func TestOnRecvPacketReturnsAckWhenPendingCleanupParseFails(t *testing.T) {
	coordinator := ibctesting.NewCoordinator(t, 1)
	chain := coordinator.GetChain(ibctesting.GetChainID(1))
	ctx := chain.GetContext()

	packet := createInvalidRecvPacket()
	middleware := ratelimiting.NewIBCMiddleware(chain.GetSimApp().RateLimitKeeper)
	expectedAck := channeltypes.NewResultAcknowledgement([]byte{1})
	middleware.SetUnderlyingApplication(mockPacketUnmarshalerModule{recvAck: expectedAck})

	ack := middleware.OnRecvPacket(ctx, "", packet, nil)
	require.Equal(t, expectedAck, ack)
}

func TestWriteAcknowledgementRemovesPendingReceivePacketForSuccessAck(t *testing.T) {
	coordinator := ibctesting.NewCoordinator(t, 1)
	chain := coordinator.GetChain(ibctesting.GetChainID(1))
	ctx := chain.GetContext()
	ratelimitKeeper := chain.GetSimApp().RateLimitKeeper
	writeAckWrapper := &mockICS4Wrapper{}
	ratelimitKeeper.SetICS4Wrapper(writeAckWrapper)

	packet, packetInfo := createRecvPacket(t)
	require.NoError(t, ratelimitKeeper.SetPendingReceivePacket(ctx, packetInfo.ChannelID, packet.Sequence, packetInfo.Denom))

	middleware := ratelimiting.NewIBCMiddleware(ratelimitKeeper)
	ack := channeltypes.NewResultAcknowledgement([]byte{1})

	err := middleware.WriteAcknowledgement(ctx, packet, ack)
	require.NoError(t, err)
	require.True(t, writeAckWrapper.writeAckCalled)

	found, err := ratelimitKeeper.CheckPacketReceivedDuringCurrentQuota(ctx, packetInfo.ChannelID, packet.Sequence, packetInfo.Denom)
	require.NoError(t, err)
	require.False(t, found)
}

func TestWriteAcknowledgementFallsBackWhenPendingCleanupParseFails(t *testing.T) {
	coordinator := ibctesting.NewCoordinator(t, 1)
	chain := coordinator.GetChain(ibctesting.GetChainID(1))
	ctx := chain.GetContext()
	writeAckWrapper := &mockICS4Wrapper{}
	chain.GetSimApp().RateLimitKeeper.SetICS4Wrapper(writeAckWrapper)

	middleware := ratelimiting.NewIBCMiddleware(chain.GetSimApp().RateLimitKeeper)
	ack := channeltypes.NewResultAcknowledgement([]byte{1})

	err := middleware.WriteAcknowledgement(ctx, createInvalidRecvPacket(), ack)
	require.NoError(t, err)
	require.True(t, writeAckWrapper.writeAckCalled)
}

func createRecvPacket(t *testing.T) (channeltypes.Packet, keeper.RateLimitedPacketInfo) {
	t.Helper()

	packetData := transfertypes.FungibleTokenPacketData{
		Denom:    "uosmo",
		Amount:   "10",
		Sender:   "sender",
		Receiver: "receiver",
	}
	packetDataBz, err := json.Marshal(packetData)
	require.NoError(t, err)

	packet := channeltypes.NewPacket(
		packetDataBz,
		testSequence,
		testTransferPort,
		"channel-1",
		testTransferPort,
		"channel-0",
		clienttypes.Height{},
		1,
	)
	packetInfo, err := keeper.ParsePacketInfo(packet, types.PACKET_RECV)
	require.NoError(t, err)

	return packet, packetInfo
}

func createInvalidRecvPacket() channeltypes.Packet {
	return channeltypes.NewPacket(
		[]byte(invalidPacketBz),
		testSequence,
		testTransferPort,
		"channel-1",
		testTransferPort,
		"channel-0",
		clienttypes.Height{},
		1,
	)
}

func setReceiveRateLimit(ctx sdk.Context, ratelimitKeeper *keeper.Keeper, packetInfo keeper.RateLimitedPacketInfo) {
	ratelimitKeeper.SetRateLimit(ctx, types.RateLimit{
		Path: &types.Path{
			Denom:             packetInfo.Denom,
			ChannelOrClientId: packetInfo.ChannelID,
		},
		Quota: &types.Quota{
			MaxPercentSend: sdkmath.NewInt(100),
			MaxPercentRecv: sdkmath.NewInt(100),
		},
		Flow: &types.Flow{
			Inflow:       sdkmath.ZeroInt(),
			Outflow:      sdkmath.ZeroInt(),
			ChannelValue: sdkmath.NewInt(100),
		},
	})
}
