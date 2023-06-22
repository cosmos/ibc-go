package ibccallback

import (
	// external libraries
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	// ibc-go
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
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
		panic("underlying application does not implement PacketDataUnmarshaler")
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

// OnAcknowledgementPacket implements wasm callbacks for acknowledgement packets.
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

	callbackData, err := types.GetCallbackData(im.app, packet, ctx.GasMeter().GasRemaining())
	if err != nil {
		types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeAcknowledgement, callbackData.ContractAddr, callbackData.GasLimit, err)
		return appResult
	}

	var ack channeltypes.Acknowledgement
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		err = errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal callback packet acknowledgement: %v", err)
		types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeAcknowledgement, callbackData.ContractAddr, callbackData.GasLimit, err)
		return appResult
	}

	err = im.contractKeeper.IBCAcknowledgementPacketCallback(ctx, packet, callbackData.CustomMsg, ack, relayer, callbackData.ContractAddr, callbackData.GasLimit)
	if err != nil {
		types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeAcknowledgement, callbackData.ContractAddr, callbackData.GasLimit, err)
		return appResult
	}

	// emit event as a callback success
	types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeAcknowledgement, callbackData.ContractAddr, callbackData.GasLimit, nil)
	return appResult
}

// OnTimeoutPacket implements the wasm callbacks for the ibc-callbacks middleware.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	appResult := im.app.OnTimeoutPacket(ctx, packet, relayer)
	if appResult != nil {
		return appResult
	}

	callbackData, err := types.GetCallbackData(im.app, packet, ctx.GasMeter().GasRemaining())
	if err != nil {
		types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeTimeout, callbackData.ContractAddr, callbackData.GasLimit, err)
		return appResult
	}

	// call the contract
	err = im.contractKeeper.IBCPacketTimeoutCallback(ctx, packet, relayer, callbackData.ContractAddr, callbackData.GasLimit)
	if err != nil {
		types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeTimeout, callbackData.ContractAddr, callbackData.GasLimit, err)
		return appResult
	}

	types.EmitSourceCallbackEvent(ctx, packet, types.CallbackTypeTimeout, callbackData.ContractAddr, callbackData.GasLimit, nil)
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

// OnRecvPacket defers to the underlying application
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	appAck := im.app.OnRecvPacket(ctx, packet, relayer)

	appAckResult, ok := appAck.(channeltypes.Acknowledgement)
	if !ok {
		return appAck
	}

	callbackData, err := types.GetCallbackData(im.app, packet, ctx.GasMeter().GasRemaining())
	if err != nil {
		types.EmitDestinationCallbackEvent(ctx, packet, types.CallbackTypeTimeout, callbackData.ContractAddr, callbackData.GasLimit, err)
		return appAck
	}

	err = im.contractKeeper.IBCReceivePacketCallback(ctx, packet, callbackData.CustomMsg, appAckResult, relayer, callbackData.ContractAddr, callbackData.GasLimit)
	if err != nil {
		types.EmitDestinationCallbackEvent(ctx, packet, types.CallbackTypeTimeout, callbackData.ContractAddr, callbackData.GasLimit, err)
		return appAck
	}
	
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
