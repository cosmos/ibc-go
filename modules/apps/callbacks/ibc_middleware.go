package ibccallbacks

import (
	"fmt"

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
	app     types.PacketUnmarshalerIBCModule
	channel porttypes.ICS4Wrapper

	contractKeeper types.ContractKeeper
}

// NewIBCMiddleware creates a new IBCMiddlware given the keeper and underlying application
func NewIBCMiddleware(
	app porttypes.IBCModule,
	channel porttypes.ICS4Wrapper,
	contractKeeper types.ContractKeeper,
) IBCMiddleware {
	packetUnmarshalerApp, ok := app.(types.PacketUnmarshalerIBCModule)
	if !ok {
		panic(fmt.Sprintf("underlying application does not implement %T", (*types.PacketUnmarshalerIBCModule)(nil)))
	}
	return IBCMiddleware{
		app:            packetUnmarshalerApp,
		channel:        channel,
		contractKeeper: contractKeeper,
	}
}

// UnmarshalPacketData defers to the underlying app to unmarshal the packet data.
func (im IBCMiddleware) UnmarshalPacketData(bz []byte) (interface{}, error) {
	return im.app.UnmarshalPacketData(bz)
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

	callbackDataGetter := func() (types.CallbackData, error) {
		return types.GetSourceCallbackData(im.app, packet.Data, ctx.GasMeter().GasRemaining())
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress string) error {
		return im.contractKeeper.IBCOnAcknowledgementPacketCallback(cachedCtx, packet, acknowledgement, relayer, callbackAddress)
	}

	im.processCallback(ctx, packet, types.CallbackTypeAcknowledgement, callbackDataGetter, callbackExecutor)
	return nil
}

// OnTimeoutPacket implements timeout source callbacks for the ibc-callbacks middleware.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback fails (within the gas limit), state changes are reverted.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	err := im.app.OnTimeoutPacket(ctx, packet, relayer)
	if err != nil {
		return err
	}

	callbackDataGetter := func() (types.CallbackData, error) {
		return types.GetSourceCallbackData(im.app, packet.Data, ctx.GasMeter().GasRemaining())
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress string) error {
		return im.contractKeeper.IBCOnTimeoutPacketCallback(cachedCtx, packet, relayer, callbackAddress)
	}

	im.processCallback(ctx, packet, types.CallbackTypeTimeoutPacket, callbackDataGetter, callbackExecutor)
	return nil
}

// OnRecvPacket implements destination callbacks for the ibc-callbacks middleware.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback fails (within the gas limit), state changes are reverted.
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	appAck := im.app.OnRecvPacket(ctx, packet, relayer)

	callbackDataGetter := func() (types.CallbackData, error) {
		return types.GetDestCallbackData(im.app, packet.Data, ctx.GasMeter().GasRemaining())
	}
	callbackExecutor := func(cachedCtx sdk.Context, callbackAddress string) error {
		return im.contractKeeper.IBCOnRecvPacketCallback(cachedCtx, packet, appAck, relayer, callbackAddress)
	}

	im.processCallback(ctx, packet, types.CallbackTypeReceivePacket, callbackDataGetter, callbackExecutor)
	return appAck
}

// processCallback executes the callbackExecutor and reverts state changes if the callbackExecutor fails.
func (im IBCMiddleware) processCallback(
	ctx sdk.Context, packet channeltypes.Packet, callbackType types.CallbackType,
	callbackDataGetter func() (types.CallbackData, error),
	callbackExecutor func(sdk.Context, string) error,
) {
	defer func() {
		if r := recover(); r != nil {
			// We handle panic here. This is to ensure that the state changes are reverted
			// and out of gas panics are handled.
			types.Logger(ctx).Info("Recovered from panic.", "panic", r)
		}
	}()

	callbackData, err := callbackDataGetter()
	if err != nil {
		types.EmitCallbackEvent(ctx, packet, callbackType, callbackData, err)
		return
	}
	if callbackData.ContractAddr == "" {
		types.Logger(ctx).Info(
			fmt.Sprintf("No %s callback found for packet.", callbackType),
			"packet", packet,
		)
		return
	}

	cachedCtx, writeFn := ctx.CacheContext()
	cachedCtx = cachedCtx.WithGasMeter(sdk.NewGasMeter(callbackData.GasLimit))

	err = callbackExecutor(cachedCtx, callbackData.ContractAddr)
	if err == nil {
		writeFn()
	}
	ctx.GasMeter().ConsumeGas(cachedCtx.GasMeter().GasConsumed(), fmt.Sprintf("ibc %s callback", callbackType))

	types.EmitCallbackEvent(ctx, packet, callbackType, callbackData, err)
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
	return im.channel.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	return im.channel.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

// GetAppVersion returns the application version of the underlying application
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.channel.GetAppVersion(ctx, portID, channelID)
}
