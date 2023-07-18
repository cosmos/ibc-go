package ibccallbacks

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var (
	_ porttypes.Middleware         = (*IBCMiddleware)(nil)
	_ porttypes.PacketInfoProvider = (*IBCMiddleware)(nil)
)

// IBCMiddleware implements the ICS26 callbacks for the ibc-callbacks middleware given
// the underlying application.
type IBCMiddleware struct {
	app         types.PacketInfoProviderIBCModule
	ics4Wrapper porttypes.ICS4Wrapper

	contractKeeper types.ContractKeeper

	maxCallbackGas uint64
}

// NewIBCMiddleware creates a new IBCMiddlware given the keeper and underlying application.
// The underlying application must implement the required callback interfaces.
func NewIBCMiddleware(
	app porttypes.IBCModule, ics4Wrapper porttypes.ICS4Wrapper,
	contractKeeper types.ContractKeeper, maxCallbackGas uint64,
) IBCMiddleware {
	packetInfoProviderApp, ok := app.(types.PacketInfoProviderIBCModule)
	if !ok {
		panic(fmt.Sprintf("underlying application does not implement %T", (*types.PacketInfoProviderIBCModule)(nil)))
	}
	return IBCMiddleware{
		app:            packetInfoProviderApp,
		ics4Wrapper:    ics4Wrapper,
		contractKeeper: contractKeeper,
		maxCallbackGas: maxCallbackGas,
	}
}

// OnAcknowledgementPacket implements source callbacks for acknowledgement packets.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback fails (within the gas limit), state changes are reverted.
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

	packetSenderAddress := im.GetPacketSender(packet)
	callbackDataGetter := func() (types.CallbackData, bool, error) {
		return types.GetSourceCallbackData(im.app, packet.Data, ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress string) error {
		return im.contractKeeper.IBCOnAcknowledgementPacketCallback(cachedCtx, packet, acknowledgement, relayer, callbackAddress, packetSenderAddress)
	}

	return im.processCallback(ctx, packet, types.CallbackTypeAcknowledgement, callbackDataGetter, callbackExecutor)
}

// OnTimeoutPacket implements timeout source callbacks for the ibc-callbacks middleware.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback fails (within the gas limit), state changes are reverted.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	err := im.app.OnTimeoutPacket(ctx, packet, relayer)
	if err != nil {
		return err
	}

	packetSenderAddress := im.GetPacketSender(packet)
	callbackDataGetter := func() (types.CallbackData, bool, error) {
		return types.GetSourceCallbackData(im.app, packet.Data, ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress string) error {
		return im.contractKeeper.IBCOnTimeoutPacketCallback(cachedCtx, packet, relayer, callbackAddress, packetSenderAddress)
	}

	return im.processCallback(ctx, packet, types.CallbackTypeTimeoutPacket, callbackDataGetter, callbackExecutor)
}

// OnRecvPacket implements the WriteAcknowledgement destination callbacks for the ibc-callbacks middleware during
// synchronous packet acknowledgement.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback fails (within the gas limit), state changes are reverted via a panic.
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	ack := im.app.OnRecvPacket(ctx, packet, relayer)
	// if ack is nil, then the callback is handled in WriteAcknowledgement
	if ack == nil {
		return nil
	}

	packetReceiverAddress := im.GetPacketReceiver(packet)
	callbackDataGetter := func() (types.CallbackData, bool, error) {
		return types.GetDestCallbackData(im.app, packet.GetData(), ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress string) error {
		return im.contractKeeper.IBCWriteAcknowledgementCallback(cachedCtx, packet, ack, callbackAddress, packetReceiverAddress)
	}

	err := im.processCallback(ctx, packet, types.CallbackTypeWriteAcknowledgement, callbackDataGetter, callbackExecutor)
	if err != nil {
		// revert entire tx if processCallback returns an error
		panic(err)
	}

	return ack
}

// WriteAcknowledgement implements the WriteAcknowledgement destination callbacks for the ibc-callbacks middleware during
// asynchronous packet acknowledgement.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback fails (within the gas limit), state changes are reverted.
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

	packetReceiverAddress := im.GetPacketReceiver(packet)
	callbackDataGetter := func() (types.CallbackData, bool, error) {
		return types.GetDestCallbackData(im.app, packet.GetData(), ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress string) error {
		return im.contractKeeper.IBCWriteAcknowledgementCallback(cachedCtx, packet, ack, callbackAddress, packetReceiverAddress)
	}

	return im.processCallback(ctx, packet, types.CallbackTypeWriteAcknowledgement, callbackDataGetter, callbackExecutor)
}

// SendPacket implements the ICS4 Wrapper interface
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

	// we use the reconstructed packet to get the packet sender, this should be fine since the only missing fields are
	// the destination port and channel. And GetPacketSender is a static method that does not depend on the context, so
	// it should be fine to use the reconstructed packet.
	reconstructedPacket := channeltypes.NewPacket(data, seq, sourcePort, sourceChannel, "", "", timeoutHeight, timeoutTimestamp)
	packetSenderAddress := im.GetPacketSender(reconstructedPacket)

	callbackDataGetter := func() (types.CallbackData, bool, error) {
		return types.GetSourceCallbackData(im.app, data, ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress string) error {
		return im.contractKeeper.IBCSendPacketCallback(
			cachedCtx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data, callbackAddress, packetSenderAddress,
		)
	}

	return seq, im.processCallback(ctx, reconstructedPacket, types.CallbackTypeSendPacket, callbackDataGetter, callbackExecutor)
}

// processCallback executes the callbackExecutor and reverts contract changes if the callbackExecutor fails.
//
// An error is returned only if the callbackExecutor panics, and the relayer has not provided enough gas.
func (im IBCMiddleware) processCallback(
	ctx sdk.Context, packet ibcexported.PacketI, callbackType types.CallbackType,
	callbackDataGetter func() (types.CallbackData, bool, error),
	callbackExecutor func(sdk.Context, string) error,
) (err error) {
	callbackData, remainingGasIsAtGasLimit, err := callbackDataGetter()
	if err != nil {
		types.EmitCallbackEvent(ctx, packet, callbackType, callbackData, err)
		return nil
	}
	if callbackData.ContractAddr == "" {
		types.Logger(ctx).Info(
			fmt.Sprintf("No %s callback found for packet.", callbackType),
			"packet", packet,
		)
		return nil
	}

	cachedCtx, writeFn := ctx.CacheContext()
	cachedCtx = cachedCtx.WithGasMeter(sdk.NewGasMeter(callbackData.GasLimit))
	defer func() {
		if r := recover(); r != nil {
			// We handle panic here. This is to ensure that the state changes are reverted
			// and out of gas panics are handled.
			if oogError, ok := r.(sdk.ErrorOutOfGas); ok {
				types.Logger(ctx).Info("Callbacks recovered from out of gas panic.", "panic", oogError)
				if !remainingGasIsAtGasLimit {
					err = errorsmod.Wrapf(types.ErrCallbackOutOfGas,
						"out of gas in location: %v; gasWanted: %d, gasUsed: %d",
						oogError.Descriptor, cachedCtx.GasMeter().Limit(), cachedCtx.GasMeter().GasConsumed(),
					)
				}
			}
		}
		ctx.GasMeter().ConsumeGas(cachedCtx.GasMeter().GasConsumed(), fmt.Sprintf("ibc %s callback", callbackType))
	}()

	err = callbackExecutor(cachedCtx, callbackData.ContractAddr)
	if err == nil {
		writeFn()
	}

	types.EmitCallbackEvent(ctx, packet, callbackType, callbackData, err)
	return nil
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

// GetAppVersion returns the application version of the underlying application
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// UnmarshalPacketData defers to the underlying app to unmarshal the packet data.
func (im IBCMiddleware) UnmarshalPacketData(bz []byte) (interface{}, error) {
	return im.app.UnmarshalPacketData(bz)
}

// GetPacketSender defers to the underlying app.
func (im IBCMiddleware) GetPacketSender(packet ibcexported.PacketI) string {
	return im.app.GetPacketSender(packet)
}

// GetPacketReceiver defers to the underlying app.
func (im IBCMiddleware) GetPacketReceiver(packet ibcexported.PacketI) string {
	return im.app.GetPacketReceiver(packet)
}
