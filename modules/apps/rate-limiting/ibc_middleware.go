package ratelimiting

import (
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	_ porttypes.Middleware              = (*IBCMiddleware)(nil)
	_ porttypes.PacketUnmarshalarModule = (*IBCMiddleware)(nil)
)

// TODO: Refactor.
// IBCMiddleware does not need explicit channelkeeper and ics4Wrapper. These 2 fields are in keeper.
//
// IBCMiddleware implements the ICS26 callbacks for the rate-limiting middleware.
type IBCMiddleware struct {
	app           porttypes.PacketUnmarshalarModule
	keeper        keeper.Keeper
	channelKeeper *channelkeeper.Keeper
	ics4Wrapper   porttypes.ICS4Wrapper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper, underlying application, and channel keeper.
func NewIBCMiddleware(app porttypes.PacketUnmarshalarModule, k keeper.Keeper, ck *channelkeeper.Keeper) IBCMiddleware {
	// The keeper needs the ICS4Wrapper to potentially send packets (though not used currently).
	// We pass the channel keeper as the ICS4Wrapper for consistency and potential future use.
	k.SetICS4Wrapper(ck)
	return IBCMiddleware{
		app:           app,
		keeper:        k,
		channelKeeper: ck,
		ics4Wrapper:   ck,
	}
}

// OnChanOpenInit implements the IBCMiddleware interface. Call underlying app's OnChanOpenInit.
func (im IBCMiddleware) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, counterparty channeltypes.Counterparty, version string) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
}

// OnChanOpenTry implements the IBCMiddleware interface. Call underlying app's OnChanOpenTry.
func (im IBCMiddleware) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCMiddleware interface. Call underlying app's OnChanOpenAck.
func (im IBCMiddleware) OnChanOpenAck(ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCMiddleware interface. Call underlying app's OnChanOpenConfirm.
func (im IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCMiddleware interface. Call underlying app's OnChanCloseInit.
func (im IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCMiddleware interface. Call underlying app's OnChanCloseConfirm.
func (im IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCMiddleware interface.
// Rate limits the incoming packet. If the packet is allowed, call underlying app's OnRecvPacket.
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	if err := im.keeper.ReceiveRateLimitedPacket(ctx, packet); err != nil {
		im.keeper.Logger(ctx).Error(fmt.Sprintf("Receive packet rate limited: %s", err.Error()))
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// If the packet was not rate-limited, pass it down to the underlying app's OnRecvPacket callback
	return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer) // Added channelVersion
}

// OnAcknowledgementPacket implements the IBCMiddleware interface.
// If the acknowledgement was an error, revert the outflow amount.
// Then, call underlying app's OnAcknowledgementPacket.
func (im IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if err := im.keeper.AcknowledgeRateLimitedPacket(ctx, packet, acknowledgement); err != nil {
		im.keeper.Logger(ctx).Error(fmt.Sprintf("Rate limit OnAcknowledgementPacket failed: %s", err.Error()))
	}

	return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCMiddleware interface.
// Revert the outflow amount. Then, call underlying app's OnTimeoutPacket.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if err := im.keeper.TimeoutRateLimitedPacket(ctx, packet); err != nil {
		im.keeper.Logger(ctx).Error(fmt.Sprintf("Rate limit OnTimeoutPacket failed: %s", err.Error()))
	}

	return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

// SendPacket implements the ICS4 Wrapper interface.
// It calls the keeper's SendRateLimitedPacket function first to check the rate limit.
// If the packet is allowed, it then calls the underlying ICS4Wrapper SendPacket.
func (im IBCMiddleware) SendPacket(ctx sdk.Context, sourcePort string, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (sequence uint64, err error) {
	if im.channelKeeper == nil {
		return 0, errors.New("channel keeper is not set on IBCMiddleware")
	}

	// Get the next sequence number from the channel keeper.
	seq, found := im.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errorsmod.Wrapf(channeltypes.ErrSequenceSendNotFound, "source port: %s, source channel: %s", sourcePort, sourceChannel)
	}

	packetToCheck := channeltypes.Packet{
		Sequence:         seq,
		SourcePort:       sourcePort,
		SourceChannel:    sourceChannel,
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
		Data:             data,
	}

	err = im.keeper.SendRateLimitedPacket(ctx, packetToCheck)
	if err != nil {
		im.keeper.Logger(ctx).Error(fmt.Sprintf("ICS20 packet send was denied by rate limiter: %s", err.Error()))
		return 0, err
	}

	seq, err = im.ics4Wrapper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		return 0, err
	}

	return seq, nil
}

// WriteAcknowledgement implements the ICS4 Wrapper interface.
// It calls the underlying ICS4Wrapper.
func (im IBCMiddleware) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return im.ics4Wrapper.WriteAcknowledgement(ctx, packet, ack)
}

// GetAppVersion implements the ICS4 Wrapper interface.
// It calls the underlying ICS4Wrapper.
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// UnmarshalPacketData implements the PacketDataUnmarshaler interface.
// It defers to the underlying app to unmarshal the packet data.
func (im IBCMiddleware) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (interface{}, string, error) {
	unmarshaler, ok := im.app.(porttypes.PacketDataUnmarshaler)
	if !ok {
		return nil, "", errorsmod.Wrapf(types.ErrUnsupportedAttribute, "underlying application does not implement %T", (*porttypes.PacketDataUnmarshaler)(nil))
	}
	return unmarshaler.UnmarshalPacketData(ctx, portID, channelID, bz)
}
