package ratelimiting

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	_ porttypes.Middleware            = (*IBCMiddleware)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCMiddleware)(nil) // Optional: If underlying app needs it
)

// IBCMiddleware implements the ICS26 callbacks for the rate-limiting middleware.
type IBCMiddleware struct {
	app         porttypes.IBCModule
	keeper      keeper.Keeper
	ics4Wrapper porttypes.ICS4Wrapper // Added: The underlying ICS4Wrapper
	// Note: We wrap the keeper with the ICS4Wrapper, which calls the underlying stack.
	// The keeper needs access to the underlying stack to send packets.
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application, and underlying ics4wrapper.
func NewIBCMiddleware(app porttypes.IBCModule, k keeper.Keeper, ics4Wrapper porttypes.ICS4Wrapper) IBCMiddleware {
	// The keeper needs the ICS4Wrapper to send packets.
	// We also store the underlying ics4wrapper directly for calls like WriteAcknowledgement and GetAppVersion.
	k.SetICS4Wrapper(ics4Wrapper) // Set the wrapper on the keeper
	return IBCMiddleware{
		app:         app,
		keeper:      k,
		ics4Wrapper: ics4Wrapper, // Store the wrapper on the middleware itself
	}
}

// SetICS4Wrapper sets the ICS4Wrapper for the middleware *and* the keeper.
// It is used after the middleware is created in app wiring because of dependency cycles.
func (im *IBCMiddleware) SetICS4Wrapper(ics4Wrapper porttypes.ICS4Wrapper) {
	im.ics4Wrapper = ics4Wrapper
	im.keeper.SetICS4Wrapper(ics4Wrapper) // Keep setting it on the keeper as well
}

// OnChanOpenInit implements the IBCMiddleware interface. Call underlying app's OnChanOpenInit.
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	// chanCap *channeltypes.Capability, // Removed: Not part of porttypes.Middleware interface
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	// Call underlying app's OnChanOpenInit
	// Pass chanCap from the IBC stack if it's required by the underlying app potentially?
	// For now, assume underlying app matches the Middleware interface without chanCap.
	// If the underlying app *needs* chanCap, this middleware might need to retrieve it.
	// Let's stick to the interface for now.
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
}

// OnChanOpenTry implements the IBCMiddleware interface. Call underlying app's OnChanOpenTry.
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	// chanCap *channeltypes.Capability, // Removed: Not part of porttypes.Middleware interface
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	// Call underlying app's OnChanOpenTry
	// See comment in OnChanOpenInit regarding chanCap
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCMiddleware interface. Call underlying app's OnChanOpenAck.
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string, // Note: Signature differs from reference, matches ibc-go v10 api.IBCModule
	counterpartyVersion string,
) error {
	// Call underlying app's OnChanOpenAck
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCMiddleware interface. Call underlying app's OnChanOpenConfirm.
func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Call underlying app's OnChanOpenConfirm
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCMiddleware interface. Call underlying app's OnChanCloseInit.
func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Call underlying app's OnChanCloseInit
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCMiddleware interface. Call underlying app's OnChanCloseConfirm.
func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Call underlying app's OnChanCloseConfirm
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCMiddleware interface.
// Rate limits the incoming packet. If the packet is allowed, call underlying app's OnRecvPacket.
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	channelVersion string, // Added: Matches porttypes.Middleware interface
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	// Check if the packet would cause the rate limit inflow quota to be exceeded
	if err := im.keeper.ReceiveRateLimitedPacket(ctx, packet); err != nil {
		im.keeper.Logger(ctx).Error(fmt.Sprintf("Receive packet rate limited: %s", err.Error()))
		// Return error acknowledgement
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// If the packet was not rate-limited, pass it down to the underlying app's OnRecvPacket callback
	return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer) // Added channelVersion
}

// OnAcknowledgementPacket implements the IBCMiddleware interface.
// If the acknowledgement was an error, revert the outflow amount.
// Then, call underlying app's OnAcknowledgementPacket.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string, // Added: Matches porttypes.Middleware interface
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// Call the keeper function to potentially revert outflow on error
	if err := im.keeper.AcknowledgeRateLimitedPacket(ctx, packet, acknowledgement); err != nil {
		// Log the error but do not block the acknowledgement processing
		im.keeper.Logger(ctx).Error(fmt.Sprintf("Rate limit OnAcknowledgementPacket failed: %s", err.Error()))
	}

	// Call underlying app's OnAcknowledgementPacket
	return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer) // Added channelVersion
}

// OnTimeoutPacket implements the IBCMiddleware interface.
// Revert the outflow amount. Then, call underlying app's OnTimeoutPacket.
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	channelVersion string, // Added: Matches porttypes.Middleware interface
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	// Call the keeper function to revert outflow
	if err := im.keeper.TimeoutRateLimitedPacket(ctx, packet); err != nil {
		// Log the error but do not block the timeout processing
		im.keeper.Logger(ctx).Error(fmt.Sprintf("Rate limit OnTimeoutPacket failed: %s", err.Error()))
	}

	// Call underlying app's OnTimeoutPacket
	return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer) // Added channelVersion
}

// SendPacket implements the ICS4 Wrapper interface.
// It calls the keeper's SendPacket function, which checks the rate limit and
// calls the underlying ICS4Wrapper.
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	sourcePort string, // Note: Signature differs from reference, matches ibc-go v10 ICS4Wrapper
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (sequence uint64, err error) {
	// Check rate limit first using the keeper's specific logic for sends
	// Note: We need a keeper method that *only* checks the limit and updates flow,
	// without calling the underlying SendPacket itself. Let's assume SendRateLimitedPacket does this for now.
	// The actual call to the underlying stack happens here.

	// Get sequence number and pass packet down the stack
	seq, err := im.ics4Wrapper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		return 0, err
	}

	// Perform rate limit check *after* getting sequence number but *before* returning
	err = im.keeper.SendRateLimitedPacket(ctx, channeltypes.Packet{
		Sequence:         seq,
		SourcePort:       sourcePort,
		SourceChannel:    sourceChannel,
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
		Data:             data,
	})
	if err != nil {
		// Packet send was denied by rate limiter, return error (sequence number is ignored by caller on error)
		im.keeper.Logger(ctx).Error(fmt.Sprintf("ICS20 packet send was denied by rate limiter: %s", err.Error()))
		return 0, err
	}

	// Packet send allowed by rate limiter
	return seq, nil
}

// WriteAcknowledgement implements the ICS4 Wrapper interface.
// It calls the underlying ICS4Wrapper.
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
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
