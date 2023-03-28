package callbacks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

type Example struct {
}

var _ porttypes.Middleware = &IBCMiddleware[Example]{}

// todo: concrete type
type CallbackPacketData any

type IBCMiddleware[T CallbackPacketData] struct {
	next     porttypes.Middleware
	decoder  Decoder[T]
	executor Executor[T]
	limits   GasLimits
}

// Decoder unpacks a raw ibc packet to the custom type
type Decoder[T CallbackPacketData] interface {
	// Decode packet to custom type. An empty result skips callback execution, error aborts process
	Decode(packet channeltypes.Packet) (*T, error)
}

// Executor executes the callback
type Executor[T CallbackPacketData] interface {
	OnRecvPacket(ctx sdk.Context, obj T, relayer sdk.AccAddress) error
	OnAcknowledgementPacket(ctx sdk.Context, obj T, acknowledgement []byte, relayer sdk.AccAddress) error
	OnTimeoutPacket(ctx sdk.Context, obj T, relayer sdk.AccAddress) error
}
type GasLimits struct {
	OnRecvPacketLimit            *sdk.Gas
	OnAcknowledgementPacketLimit *sdk.Gas
	OnTimeoutPacketLimit         *sdk.Gas
}

// NewIBCMiddleware constructor
func NewIBCMiddleware[T CallbackPacketData](next porttypes.Middleware, decoder Decoder[T], executor Executor[T], limits GasLimits) IBCMiddleware[T] {
	return IBCMiddleware[T]{
		next:     next,
		decoder:  decoder,
		executor: executor,
		limits:   limits,
	}
}

// OnRecvPacket implements the IBCMiddleware interface.
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware[T]) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) exported.Acknowledgement {

	ack := im.next.OnRecvPacket(ctx, packet, relayer)

	// todo: limit gas
	p, err := im.decoder.Decode(packet)
	if err != nil {
		// see comments in https://github.com/cosmos/ibc-go/blob/main/docs/architecture/adr-008-app-caller-cbs/adr-008-app-caller-cbs.md
		// todo: filter out events: https://github.com/cosmos/ibc-go/issues/3358
		// todo: set event with raw error for debug as defined in ErrorAcknowledgement?
		return channeltypes.NewErrorAcknowledgement(err)
	}
	if p != nil {
		if err := im.executor.OnRecvPacket(ctx, *p, relayer); err != nil {
			// see comments in https://github.com/cosmos/ibc-go/blob/main/docs/architecture/adr-008-app-caller-cbs/adr-008-app-caller-cbs.md
			// todo: filter out events: https://github.com/cosmos/ibc-go/issues/3358
			// todo: set event with raw error for debug as defined in ErrorAcknowledgement?
			return channeltypes.NewErrorAcknowledgement(err)
		}
	}
	return ack
}

// OnAcknowledgementPacket implements the IBCMiddleware interface
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware[T]) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// call underlying callback
	if err := im.next.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer); err != nil {
		return err
	}
	// todo: limit gas
	p, err := im.decoder.Decode(packet)
	if err != nil {
		return err
	}
	if p != nil {
		return im.executor.OnAcknowledgementPacket(ctx, *p, acknowledgement, relayer)
	}
	return nil
}

// OnTimeoutPacket implements the IBCMiddleware interface
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware[T]) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	if err := im.next.OnTimeoutPacket(ctx, packet, relayer); err != nil {
		return err
	}
	// todo: limit gas
	p, err := im.decoder.Decode(packet)
	if err != nil {
		return err
	}
	if p != nil {
		return im.executor.OnTimeoutPacket(ctx, *p, relayer)
	}
	return nil
}

// OnChanOpenInit implements the IBCMiddleware interface
func (im IBCMiddleware[T]) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return im.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
}

// OnChanOpenTry implements the IBCMiddleware interface
func (im IBCMiddleware[T]) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return im.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCMiddleware interface
func (im IBCMiddleware[T]) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return im.next.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCMiddleware interface
func (im IBCMiddleware[T]) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.next.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCMiddleware interface
func (im IBCMiddleware[T]) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.next.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCMiddleware interface
func (im IBCMiddleware[T]) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.next.OnChanCloseConfirm(ctx, portID, channelID)
}

// SendPacket implements the ICS4 Wrapper interface
func (im IBCMiddleware[T]) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	return im.next.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware[T]) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
	ack exported.Acknowledgement,
) error {
	return im.next.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

// GetAppVersion returns the application version of the underlying application
func (im IBCMiddleware[T]) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return "ics-8.0-alex", true
}
