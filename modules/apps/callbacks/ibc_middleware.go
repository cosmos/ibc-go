package ibccallbacks

import (
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
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

// SendPacket implements source callbacks for sending packets.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback returns an error, panics, or runs out of gas, then
// the packet send is rejected.
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
		return 0, err
	}

	callbackData, err := types.GetSourceCallbackData(im.app, data, sourcePort, ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	// SendPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return seq, nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCSendPacketCallback(
			cachedCtx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data, callbackData.CallbackAddress, callbackData.SenderAddress,
		)
	}

	err = im.processCallback(ctx, types.CallbackTypeSendPacket, callbackData, callbackExecutor)
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
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// we first call the underlying app to handle the acknowledgement
	err := im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
	if err != nil {
		return err
	}

	callbackData, err := types.GetSourceCallbackData(
		im.app, packet.GetData(), packet.GetSourcePort(), ctx.GasMeter().GasRemaining(), im.maxCallbackGas,
	)
	// OnAcknowledgementPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCOnAcknowledgementPacketCallback(
			cachedCtx, packet, acknowledgement, relayer, callbackData.CallbackAddress, callbackData.SenderAddress,
		)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = im.processCallback(ctx, types.CallbackTypeAcknowledgementPacket, callbackData, callbackExecutor)
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
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	err := im.app.OnTimeoutPacket(ctx, packet, relayer)
	if err != nil {
		return err
	}

	callbackData, err := types.GetSourceCallbackData(
		im.app, packet.GetData(), packet.GetSourcePort(), ctx.GasMeter().GasRemaining(), im.maxCallbackGas,
	)
	// OnTimeoutPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCOnTimeoutPacketCallback(cachedCtx, packet, relayer, callbackData.CallbackAddress, callbackData.SenderAddress)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = im.processCallback(ctx, types.CallbackTypeTimeoutPacket, callbackData, callbackExecutor)
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
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	ack := im.app.OnRecvPacket(ctx, packet, relayer)
	// if ack is nil (asynchronous acknowledgements), then the callback will be handled in WriteAcknowledgement
	// if ack is not successful, all state changes are reverted. If a packet cannot be received, then there is
	// no need to execute a callback on the receiving chain.
	if ack == nil || !ack.Success() {
		return ack
	}

	callbackData, err := types.GetDestCallbackData(
		im.app, packet.GetData(), packet.GetSourcePort(), ctx.GasMeter().GasRemaining(), im.maxCallbackGas,
	)
	// OnRecvPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return ack
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCReceivePacketCallback(cachedCtx, packet, ack, callbackData.CallbackAddress)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = im.processCallback(ctx, types.CallbackTypeReceivePacket, callbackData, callbackExecutor)
	types.EmitCallbackEvent(
		ctx, packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
		types.CallbackTypeReceivePacket, callbackData, err,
	)

	return ack
}

// WriteAcknowledgement implements the ReceivePacket destination callbacks for the ibc-callbacks middleware
// during asynchronous packet acknowledgement.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
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

	callbackData, err := types.GetDestCallbackData(
		im.app, packet.GetData(), packet.GetSourcePort(), ctx.GasMeter().GasRemaining(), im.maxCallbackGas,
	)
	// WriteAcknowledgement is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCReceivePacketCallback(cachedCtx, packet, ack, callbackData.CallbackAddress)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = im.processCallback(ctx, types.CallbackTypeReceivePacket, callbackData, callbackExecutor)
	types.EmitCallbackEvent(
		ctx, packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
		types.CallbackTypeReceivePacket, callbackData, err,
	)

	return nil
}

// processCallback executes the callbackExecutor and reverts contract changes if the callbackExecutor fails.
//
// Error Precedence and Returns:
//   - oogErr: Takes the highest precedence. If the callback runs out of gas, an error wrapped with types.ErrCallbackOutOfGas is returned.
//   - panicErr: Takes the second-highest precedence. If a panic occurs and it is not propagated, an error wrapped with types.ErrCallbackPanic is returned.
//   - callbackErr: If the callbackExecutor returns an error, it is returned as-is.
//
// panics if
//   - the contractExecutor panics for any reason, and the callbackType is SendPacket, or
//   - the contractExecutor runs out of gas and the relayer has not reserved gas grater than or equal to
//     CommitGasLimit.
func (IBCMiddleware) processCallback(
	ctx sdk.Context, callbackType types.CallbackType,
	callbackData types.CallbackData, callbackExecutor func(sdk.Context) error,
) (err error) {
	cachedCtx, writeFn := ctx.CacheContext()
	cachedCtx = cachedCtx.WithGasMeter(storetypes.NewGasMeter(callbackData.ExecutionGasLimit))

	defer func() {
		// consume the minimum of g.consumed and g.limit
		ctx.GasMeter().ConsumeGas(cachedCtx.GasMeter().GasConsumedToLimit(), fmt.Sprintf("ibc %s callback", callbackType))

		// recover from all panics except during SendPacket callbacks
		if r := recover(); r != nil {
			if callbackType == types.CallbackTypeSendPacket {
				panic(r)
			}
			err = errorsmod.Wrapf(types.ErrCallbackPanic, "ibc %s callback panicked with: %v", callbackType, r)
		}

		// if the callback ran out of gas and the relayer has not reserved enough gas, then revert the state
		if cachedCtx.GasMeter().IsPastLimit() {
			if callbackData.AllowRetry() {
				panic(storetypes.ErrorOutOfGas{Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", callbackType, callbackData.CommitGasLimit)})
			}
			err = errorsmod.Wrapf(types.ErrCallbackOutOfGas, "ibc %s callback out of gas", callbackType)
		}

		// allow the transaction to be committed, continuing the packet lifecycle
	}()

	err = callbackExecutor(cachedCtx)
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
