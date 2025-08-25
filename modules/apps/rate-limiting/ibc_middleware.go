package ratelimiting

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	_ porttypes.Middleware              = (*IBCMiddleware)(nil)
	_ porttypes.PacketUnmarshalerModule = (*IBCMiddleware)(nil)
)

// IBCMiddleware implements the ICS26 callbacks for the rate-limiting middleware.
type IBCMiddleware struct {
	app    porttypes.PacketUnmarshalerModule
	keeper *keeper.Keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper, underlying application, and channel keeper.
func NewIBCMiddleware(k *keeper.Keeper) *IBCMiddleware {
	return &IBCMiddleware{
		keeper: k,
	}
}

// OnChanOpenInit implements the IBCMiddleware interface. Call underlying app's OnChanOpenInit.
func (im *IBCMiddleware) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, counterparty channeltypes.Counterparty, version string) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
}

// OnChanOpenTry implements the IBCMiddleware interface. Call underlying app's OnChanOpenTry.
func (im *IBCMiddleware) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCMiddleware interface. Call underlying app's OnChanOpenAck.
func (im *IBCMiddleware) OnChanOpenAck(ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCMiddleware interface. Call underlying app's OnChanOpenConfirm.
func (im *IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCMiddleware interface. Call underlying app's OnChanCloseInit.
func (im *IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCMiddleware interface. Call underlying app's OnChanCloseConfirm.
func (im *IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCMiddleware interface.
// Rate limits the incoming packet. If the packet is allowed, call underlying app's OnRecvPacket.
func (im *IBCMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	if err := im.keeper.ReceiveRateLimitedPacket(ctx, packet); err != nil {
		im.keeper.Logger(ctx).Error("Receive packet rate limited", "error", err)
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// If the packet was not rate-limited, pass it down to the underlying app's OnRecvPacket callback
	return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

// OnAcknowledgementPacket implements the IBCMiddleware interface.
// If the acknowledgement was an error, revert the outflow amount.
// Then, call underlying app's OnAcknowledgementPacket.
func (im *IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if err := im.keeper.AcknowledgeRateLimitedPacket(ctx, packet, acknowledgement); err != nil {
		im.keeper.Logger(ctx).Error("Rate limit OnAcknowledgementPacket failed", "error", err)
	}

	return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCMiddleware interface.
// Revert the outflow amount. Then, call underlying app's OnTimeoutPacket.
func (im *IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if err := im.keeper.TimeoutRateLimitedPacket(ctx, packet); err != nil {
		im.keeper.Logger(ctx).Error("Rate limit OnTimeoutPacket failed", "error", err)
	}

	return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

// SendPacket implements the ICS4 Wrapper interface.
// It calls the keeper's SendRateLimitedPacket function first to check the rate limit.
// If the packet is allowed, it then calls the underlying ICS4Wrapper SendPacket.
func (im *IBCMiddleware) SendPacket(ctx sdk.Context, sourcePort string, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error) {
	err := im.keeper.SendRateLimitedPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		im.keeper.Logger(ctx).Error("ICS20 packet send was denied by rate limiter", "error", err)
		return 0, err
	}

	return im.keeper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement implements the ICS4 Wrapper interface.
// It calls the underlying ICS4Wrapper.
func (im *IBCMiddleware) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return im.keeper.WriteAcknowledgement(ctx, packet, ack)
}

// GetAppVersion implements the ICS4 Wrapper interface.
// It calls the underlying ICS4Wrapper.
func (im *IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.keeper.GetAppVersion(ctx, portID, channelID)
}

// UnmarshalPacketData implements the PacketDataUnmarshaler interface.
// It defers to the underlying app to unmarshal the packet data.
func (im *IBCMiddleware) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (any, string, error) {
	return im.app.UnmarshalPacketData(ctx, portID, channelID, bz)
}

func (im *IBCMiddleware) SetICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	if wrapper == nil {
		panic("ICS4Wrapper cannot be nil")
	}
	im.keeper.SetICS4Wrapper(wrapper)
}

func (im *IBCMiddleware) SetUnderlyingApplication(app porttypes.IBCModule) {
	if im.app != nil {
		panic("underlying application already set")
	}
	// the underlying application must implement the PacketUnmarshalerModule interface
	pdApp, ok := app.(porttypes.PacketUnmarshalerModule)
	if !ok {
		panic(fmt.Errorf("underlying application must implement PacketUnmarshalerModule, got %T", app))
	}
	im.app = pdApp
}
