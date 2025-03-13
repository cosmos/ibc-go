package ibccallbacks

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/internal"
	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	_ porttypes.Middleware            = (*IBCMiddleware)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCMiddleware)(nil)
)

// IBCMiddleware implements the ICS26 callbacks for the ibc-callbacks middleware given
// the underlying application.
type IBCMiddleware struct {
	app         types.CallbacksCompatibleModule
	ics4Wrapper porttypes.ICS4Wrapper

	contractKeeper types.ContractKeeper

	// maxCallbackGas defines the maximum amount of gas that a callback actor can ask the
	// relayer to pay for. If a callback fails due to insufficient gas, the entire tx
	// is reverted if the relayer hadn't provided the minimum(userDefinedGas, maxCallbackGas).
	// If the actor hasn't defined a gas limit, then it is assumed to be the maxCallbackGas.
	maxCallbackGas uint64
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application.
// The underlying application must implement the required callback interfaces.
func NewIBCMiddleware(
	app porttypes.IBCModule, ics4Wrapper porttypes.ICS4Wrapper,
	contractKeeper types.ContractKeeper, maxCallbackGas uint64,
) IBCMiddleware {
	packetDataUnmarshalerApp, ok := app.(types.CallbacksCompatibleModule)
	if !ok {
		panic(fmt.Errorf("underlying application does not implement %T", (*types.CallbacksCompatibleModule)(nil)))
	}

	if ics4Wrapper == nil {
		panic(errors.New("ICS4Wrapper cannot be nil"))
	}

	if contractKeeper == nil {
		panic(errors.New("contract keeper cannot be nil"))
	}

	if maxCallbackGas == 0 {
		panic(errors.New("maxCallbackGas cannot be zero"))
	}

	return IBCMiddleware{
		app:            packetDataUnmarshalerApp,
		ics4Wrapper:    ics4Wrapper,
		contractKeeper: contractKeeper,
		maxCallbackGas: maxCallbackGas,
	}
}

// WithICS4Wrapper sets the ICS4Wrapper. This function may be used after the
// middleware's creation to set the middleware which is above this module in
// the IBC application stack.
func (im *IBCMiddleware) WithICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	im.ics4Wrapper = wrapper
}

// GetICS4Wrapper returns the ICS4Wrapper.
func (im *IBCMiddleware) GetICS4Wrapper() porttypes.ICS4Wrapper {
	return im.ics4Wrapper
}

// SendPacket implements source callbacks for sending packets.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback returns an error, panics, or runs out of gas, then
// the packet send is rejected.
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	seq, err := im.ics4Wrapper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		return 0, err
	}

	// packet is created without destination information present, GetSourceCallbackData does not use these.
	packet := channeltypes.NewPacket(data, seq, sourcePort, sourceChannel, "", "", timeoutHeight, timeoutTimestamp)

	callbackData, isCbPacket, err := types.GetSourceCallbackData(ctx, im.app, packet, im.maxCallbackGas)
	// SendPacket is not blocked if the packet does not opt-in to callbacks
	if !isCbPacket {
		return seq, nil
	}
	// if the packet does opt-in to callbacks but the callback data is malformed,
	// then the packet send is rejected.
	if err != nil {
		return 0, err
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCSendPacketCallback(
			cachedCtx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data, callbackData.CallbackAddress, callbackData.SenderAddress, callbackData.ApplicationVersion,
		)
	}

	err = internal.ProcessCallback(ctx, types.CallbackTypeSendPacket, callbackData, callbackExecutor)
	// contract keeper is allowed to reject the packet send.
	if err != nil {
		return 0, err
	}

	types.EmitCallbackEvent(ctx, sourcePort, sourceChannel, seq, types.CallbackTypeSendPacket, callbackData, nil)
	return seq, nil
}

// OnAcknowledgementPacket implements source callbacks for acknowledgement packets.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// we first call the underlying app to handle the acknowledgement
	err := im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
	if err != nil {
		return err
	}

	callbackData, isCbPacket, err := types.GetSourceCallbackData(
		ctx, im.app, packet, im.maxCallbackGas,
	)
	// OnAcknowledgementPacket is not blocked if the packet does not opt-in to callbacks
	if !isCbPacket {
		return nil
	}
	// if the packet does opt-in to callbacks but the callback data is malformed,
	// then the packet acknowledgement is rejected.
	// This should never occur, since this error is already checked on `SendPacket`
	if err != nil {
		return err
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCOnAcknowledgementPacketCallback(
			cachedCtx, packet, acknowledgement, relayer, callbackData.CallbackAddress, callbackData.SenderAddress, callbackData.ApplicationVersion,
		)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = internal.ProcessCallback(ctx, types.CallbackTypeAcknowledgementPacket, callbackData, callbackExecutor)
	types.EmitCallbackEvent(
		ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(),
		types.CallbackTypeAcknowledgementPacket, callbackData, err,
	)

	return nil
}

// OnTimeoutPacket implements timeout source callbacks for the ibc-callbacks middleware.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	err := im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
	if err != nil {
		return err
	}

	callbackData, isCbPacket, err := types.GetSourceCallbackData(
		ctx, im.app, packet, im.maxCallbackGas,
	)
	// OnTimeoutPacket is not blocked if the packet does not opt-in to callbacks
	if !isCbPacket {
		return nil
	}
	// if the packet does opt-in to callbacks but the callback data is malformed,
	// then the packet timeout is rejected.
	// This should never occur, since this error is already checked on `SendPacket`
	if err != nil {
		return err
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCOnTimeoutPacketCallback(cachedCtx, packet, relayer, callbackData.CallbackAddress, callbackData.SenderAddress, callbackData.ApplicationVersion)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = internal.ProcessCallback(ctx, types.CallbackTypeTimeoutPacket, callbackData, callbackExecutor)
	types.EmitCallbackEvent(
		ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(),
		types.CallbackTypeTimeoutPacket, callbackData, err,
	)

	return nil
}

// OnRecvPacket implements the ReceivePacket destination callbacks for the ibc-callbacks middleware during
// synchronous packet acknowledgement.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	ack := im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
	// if ack is nil (asynchronous acknowledgements), then the callback will be handled in WriteAcknowledgement
	// if ack is not successful, all state changes are reverted. If a packet cannot be received, then there is
	// no need to execute a callback on the receiving chain.
	if ack == nil || !ack.Success() {
		return ack
	}

	callbackData, isCbPacket, err := types.GetDestCallbackData(
		ctx, im.app, packet, im.maxCallbackGas,
	)
	// OnRecvPacket is not blocked if the packet does not opt-in to callbacks
	if !isCbPacket {
		return ack
	}
	// if the packet does opt-in to callbacks but the callback data is malformed,
	// then the packet receive is rejected.
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCReceivePacketCallback(cachedCtx, packet, ack, callbackData.CallbackAddress, callbackData.ApplicationVersion)
	}

	// callback execution errors in RecvPacket are allowed to write an error acknowledgement
	// in this case, the receive logic of the underlying app is reverted
	// and the error acknowledgement is processed on the sending chain
	// Thus the sending application MUST be capable of processing the standard channel acknowledgement
	err = internal.ProcessCallback(ctx, types.CallbackTypeReceivePacket, callbackData, callbackExecutor)
	types.EmitCallbackEvent(
		ctx, packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
		types.CallbackTypeReceivePacket, callbackData, err,
	)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return ack
}

// WriteAcknowledgement implements the ReceivePacket destination callbacks for the ibc-callbacks middleware
// during asynchronous packet acknowledgement.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	err := im.ics4Wrapper.WriteAcknowledgement(ctx, packet, ack)
	if err != nil {
		return err
	}

	chanPacket, ok := packet.(channeltypes.Packet)
	if !ok {
		panic(fmt.Errorf("expected type %T, got %T", &channeltypes.Packet{}, packet))
	}

	callbackData, isCbPacket, err := types.GetDestCallbackData(
		ctx, im.app, chanPacket, im.maxCallbackGas,
	)
	// WriteAcknowledgement is not blocked if the packet does not opt-in to callbacks
	if !isCbPacket {
		return nil
	}
	// This should never occur, since this error is already checked on `OnRecvPacket`
	if err != nil {
		return err
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCReceivePacketCallback(cachedCtx, packet, ack, callbackData.CallbackAddress, callbackData.ApplicationVersion)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = internal.ProcessCallback(ctx, types.CallbackTypeReceivePacket, callbackData, callbackExecutor)
	types.EmitCallbackEvent(
		ctx, packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
		types.CallbackTypeReceivePacket, callbackData, err,
	)

	return nil
}

// OnChanOpenInit defers to the underlying application
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	channelOrdering channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return im.app.OnChanOpenInit(ctx, channelOrdering, connectionHops, portID, channelID, counterparty, version)
}

// OnChanOpenTry defers to the underlying application
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	channelOrdering channeltypes.Order,
	connectionHops []string, portID,
	channelID string,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return im.app.OnChanOpenTry(ctx, channelOrdering, connectionHops, portID, channelID, counterparty, counterpartyVersion)
}

// OnChanOpenAck defers to the underlying application
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID,
	counterpartyChannelID,
	counterpartyVersion string,
) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm defers to the underlying application
func (im IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit defers to the underlying application
func (im IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm defers to the underlying application
func (im IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// GetAppVersion implements the ICS4Wrapper interface. Callbacks has no version,
// so the call is deferred to the underlying application.
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// UnmarshalPacketData defers to the underlying app to unmarshal the packet data.
// This function implements the optional PacketDataUnmarshaler interface.
func (im IBCMiddleware) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (interface{}, string, error) {
	return im.app.UnmarshalPacketData(ctx, portID, channelID, bz)
}
