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
	appResult := im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
	if appResult != nil {
		return appResult
	}

	callbackData, err := types.GetSourceCallbackData(im.app, packet, ctx.GasMeter().GasRemaining())
	if err != nil {
		types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeAcknowledgement, callbackData, err)
		return appResult
	}
	if callbackData.ContractAddr == "" {
		return appResult
	}

	cachedCtx, writeFn := ctx.CacheContext()
	cachedCtx = cachedCtx.WithGasMeter(sdk.NewGasMeter(callbackData.GasLimit))

	err = im.contractKeeper.IBCAcknowledgementPacketCallback(cachedCtx, packet, acknowledgement, relayer, callbackData.ContractAddr)
	if err == nil {
		writeFn()
	}
	// consume gas
	ctx.GasMeter().ConsumeGas(cachedCtx.GasMeter().GasConsumed(), "ibc acknowledgement packet callback")

	// emit event as a callback success
	types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeAcknowledgement, callbackData, err)
	return appResult
}

// OnTimeoutPacket implements timeout source callbacks for the ibc-callbacks middleware.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback fails (within the gas limit), state changes are reverted.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	appResult := im.app.OnTimeoutPacket(ctx, packet, relayer)
	if appResult != nil {
		return appResult
	}

	callbackData, err := types.GetSourceCallbackData(im.app, packet, ctx.GasMeter().GasRemaining())
	if err != nil {
		types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeTimeoutPacket, callbackData, err)
		return appResult
	}
	if callbackData.ContractAddr == "" {
		return appResult
	}

	cachedCtx, writeFn := ctx.CacheContext()
	cachedCtx = cachedCtx.WithGasMeter(sdk.NewGasMeter(callbackData.GasLimit))

	// call the contract
	err = im.contractKeeper.IBCPacketTimeoutCallback(cachedCtx, packet, relayer, callbackData.ContractAddr)
	if err == nil {
		writeFn()
	}
	ctx.GasMeter().ConsumeGas(cachedCtx.GasMeter().GasConsumed(), "ibc packet timeout callback")

	types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeTimeoutPacket, callbackData, err)
	return appResult
}

// OnChanCloseConfirm defers to the underlying application
func (im IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnChanCloseInit defers to the underlying application
func (im IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
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

// OnRecvPacket implements destination callbacks for the ibc-callbacks middleware.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback fails (within the gas limit), state changes are reverted.
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	appAck := im.app.OnRecvPacket(ctx, packet, relayer)

	callbackData, err := types.GetDestCallbackData(im.app, packet, ctx.GasMeter().GasRemaining())
	if err != nil {
		types.EmitDestinationCallbackEvent(ctx, packet, types.CallbackTypeTimeoutPacket, callbackData, err)
		return appAck
	}
	if callbackData.ContractAddr == "" {
		return appAck
	}

	cachedCtx, writeFn := ctx.CacheContext()
	cachedCtx = cachedCtx.WithGasMeter(sdk.NewGasMeter(callbackData.GasLimit))

	err = im.contractKeeper.IBCReceivePacketCallback(cachedCtx, packet, appAck.Acknowledgement(), relayer, callbackData.ContractAddr)
	if err == nil {
		writeFn()
	}
	ctx.GasMeter().ConsumeGas(cachedCtx.GasMeter().GasConsumed(), "ibc receive packet callback")

	types.EmitDestinationCallbackEvent(ctx, packet, types.CallbackTypeTimeoutPacket, callbackData, err)
	return appAck
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
func (m IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return m.channel.GetAppVersion(ctx, portID, channelID)
}
