package ibccallbacks

import (
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
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

// NewIBCMiddleware creates a new IBCMiddlware given the keeper and underlying application.
// The underlying application must implement the required callback interfaces.
func NewIBCMiddleware(
	app porttypes.IBCModule, ics4Wrapper porttypes.ICS4Wrapper,
	contractKeeper types.ContractKeeper, maxCallbackGas uint64,
) IBCMiddleware {
	packetDataUnmarshalerApp, ok := app.(types.CallbacksCompatibleModule)
	if !ok {
		panic(fmt.Sprintf("underlying application does not implement %T", (*types.CallbacksCompatibleModule)(nil)))
	}
	return IBCMiddleware{
		app:            packetDataUnmarshalerApp,
		ics4Wrapper:    ics4Wrapper,
		contractKeeper: contractKeeper,
		maxCallbackGas: maxCallbackGas,
	}
}

// SendPacket implements source callbacks for sending packets.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit
// then the state changes are reverted.
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	seq, err := im.ics4Wrapper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		return seq, err
	}

	// Reconstruct the sent packet. The destination portID and channelID are intentionally left empty as the sender information
	// is only derived from the source packet information in `GetSourceCallbackData`.
	reconstructedPacket := channeltypes.NewPacket(data, seq, sourcePort, sourceChannel, "", "", timeoutHeight, timeoutTimestamp)

	callbackDataGetter := func() (types.CallbackData, bool, error) {
		return types.GetSourceCallbackData(im.app, reconstructedPacket, ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress, packetSenderAddress string) error {
		return im.contractKeeper.IBCSendPacketCallback(
			cachedCtx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data, callbackAddress, packetSenderAddress,
		)
	}

	err = im.processCallback(ctx, reconstructedPacket, types.CallbackTypeSendPacket, callbackDataGetter, callbackExecutor)
	// contract keeper is allowed to reject the packet send.
	if err != nil {
		return 0, err
	}

	return seq, nil
}

// OnAcknowledgementPacket implements source callbacks for acknowledgement packets.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit
// then the state changes are reverted.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// we first call the underlying app to handle the acknowledgement
	err := im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
	if err != nil {
		return err
	}

	callbackDataGetter := func() (types.CallbackData, bool, error) {
		return types.GetSourceCallbackData(im.app, packet, ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress, packetSenderAddress string) error {
		return im.contractKeeper.IBCOnAcknowledgementPacketCallback(cachedCtx, packet, acknowledgement, relayer, callbackAddress, packetSenderAddress)
	}

	err = im.processCallback(ctx, packet, types.CallbackTypeAcknowledgement, callbackDataGetter, callbackExecutor)
	if errors.Is(err, types.ErrCallbackOutOfGas) {
		return err
	}

	return nil
}

// OnTimeoutPacket implements timeout source callbacks for the ibc-callbacks middleware.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit
// then the state changes are reverted.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	err := im.app.OnTimeoutPacket(ctx, packet, relayer)
	if err != nil {
		return err
	}

	callbackDataGetter := func() (types.CallbackData, bool, error) {
		return types.GetSourceCallbackData(im.app, packet, ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress, packetSenderAddress string) error {
		return im.contractKeeper.IBCOnTimeoutPacketCallback(cachedCtx, packet, relayer, callbackAddress, packetSenderAddress)
	}

	err = im.processCallback(ctx, packet, types.CallbackTypeTimeoutPacket, callbackDataGetter, callbackExecutor)
	if errors.Is(err, types.ErrCallbackOutOfGas) {
		return err
	}

	return nil
}

// OnRecvPacket implements the WriteAcknowledgement destination callbacks for the ibc-callbacks middleware during
// synchronous packet acknowledgement.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	ack := im.app.OnRecvPacket(ctx, packet, relayer)
	// if ack is nil (asynchronous acknowledgements), then the callback will be handled in WriteAcknowledgement
	// if ack is not successful, all state changes are reverted. If a packet cannot be received, then you need not
	// execute a callback on the receiving chain.
	if ack == nil || !ack.Success() {
		return ack
	}

	callbackDataGetter := func() (types.CallbackData, bool, error) {
		return types.GetDestCallbackData(im.app, packet, ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress, _ string) error {
		return im.contractKeeper.IBCWriteAcknowledgementCallback(cachedCtx, packet, ack, callbackAddress)
	}

	err := im.processCallback(ctx, packet, types.CallbackTypeWriteAcknowledgement, callbackDataGetter, callbackExecutor)
	if errors.Is(err, types.ErrCallbackOutOfGas) {
		// revert entire tx if processCallback returns ErrCallbackOutOfGas, requiring the relayer to provide more gas on a retry
		panic(err)
	}

	return ack
}

// WriteAcknowledgement implements the WriteAcknowledgement destination callbacks for the ibc-callbacks middleware
// during asynchronous packet acknowledgement.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit
// then the state changes are reverted.
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	err := im.ics4Wrapper.WriteAcknowledgement(ctx, chanCap, packet, ack)
	if err != nil {
		return err
	}

	callbackDataGetter := func() (types.CallbackData, bool, error) {
		return types.GetDestCallbackData(im.app, packet, ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress, _ string) error {
		return im.contractKeeper.IBCWriteAcknowledgementCallback(cachedCtx, packet, ack, callbackAddress)
	}

	err = im.processCallback(ctx, packet, types.CallbackTypeWriteAcknowledgement, callbackDataGetter, callbackExecutor)
	if errors.Is(err, types.ErrCallbackOutOfGas) {
		return err
	}

	return nil
}

// processCallback executes the callbackExecutor and reverts contract changes if the callbackExecutor fails.
//
// An error is returned only if the callbackExecutor panics, and the relayer has not provided enough gas.
func (im IBCMiddleware) processCallback(
	ctx sdk.Context, packet ibcexported.PacketI, callbackType types.CallbackType,
	callbackDataGetter func() (types.CallbackData, bool, error),
	callbackExecutor func(sdk.Context, string, string) error,
) (err error) {
	callbackData, allowRetry, err := callbackDataGetter()
	if err != nil {
		types.Logger(ctx).Debug("Failed to get callback data.", "packet", packet, "err", err)
		return nil
	}
	if callbackData.ContractAddr == "" {
		types.Logger(ctx).Debug(fmt.Sprintf("No %s callback found for packet.", callbackType), "packet", packet)
		return nil
	}

	cachedCtx, writeFn := ctx.CacheContext()
	cachedCtx = cachedCtx.WithGasMeter(sdk.NewGasMeter(callbackData.GasLimit))
	defer func() {
		if r := recover(); r != nil {
			// We handle panic here. This is to ensure that the state changes are reverted
			// and out of gas panics are handled.
			if oogError, ok := r.(sdk.ErrorOutOfGas); ok {
				types.Logger(ctx).Debug("Callbacks recovered from out of gas panic.", "packet", packet, "panic", oogError)
				if allowRetry {
					err = errorsmod.Wrapf(types.ErrCallbackOutOfGas,
						"out of gas in location: %v; gasWanted: %d, gasUsed: %d",
						oogError.Descriptor, cachedCtx.GasMeter().Limit(), cachedCtx.GasMeter().GasConsumed(),
					)
				}
			}
		}
		types.EmitCallbackEvent(ctx, packet, callbackType, callbackData, err)
		ctx.GasMeter().ConsumeGas(cachedCtx.GasMeter().GasConsumedToLimit(), fmt.Sprintf("ibc %s callback", callbackType))
	}()

	err = callbackExecutor(cachedCtx, callbackData.ContractAddr, callbackData.AuthAddr)
	if err == nil {
		writeFn()
	}

	return err
}

// OnChanOpenInit defers to the underlying application
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	channelOrdering channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return im.app.OnChanOpenInit(ctx, channelOrdering, connectionHops, portID, channelID, channelCap, counterparty, version)
}

// OnChanOpenTry defers to the underlying application
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	channelOrdering channeltypes.Order,
	connectionHops []string, portID,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return im.app.OnChanOpenTry(ctx, channelOrdering, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion)
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
func (im IBCMiddleware) UnmarshalPacketData(bz []byte) (interface{}, error) {
	return im.app.UnmarshalPacketData(bz)
}
