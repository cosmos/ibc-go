# Scaffold for ICS20 Callback Middleware

The following is a very simple scaffold for middleware that implements actor callbacks for transfer channels. Since a channel does not need to be owned by a single actor, the handshake callbacks are no-ops. The packet callbacks will call into the relevant actor. For `OnRecvPacket`, this will be `data.Receiver`, and for `OnAcknowledgePacket` and `OnTimeoutPacket` this will be `data.Sender`.

The exact nature of the callbacks to the smart contract will depend on the environment in question (e.g. cosmwasm, evm). Thus the place where the callbacks to the smart contract are stubbed out and commented so that it can be completed by implementers for the specific environment they are targetting.

Implementers may wish to support callbacks to more IBC applications by adding a switch statement to unmarshal the specific packet data types they wish to support and passing them into the smart contract callback functions.

### Scaffold Middleware

```go
package callbacks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v6/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
)

var _ porttypes.Middleware = &IBCMiddleware{}

// IBCMiddleware implements the ICS26 callbacks for the fee middleware given the
// fee keeper and the underlying application.
type IBCMiddleware struct {
	app         porttypes.IBCModule
	chanWrapper porttypes.ICS4Wrapper
}

// NewIBCMiddleware creates a new IBCMiddlware given the keeper and underlying application
func NewIBCMiddleware(app porttypes.IBCModule, chanWrapper porttypes.ICS4Wrapper) IBCMiddleware {
	return IBCMiddleware{
		app:         app,
		chanWrapper: chanWrapper,
	}
}

// OnChanOpenInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	// call underlying app's OnChanOpenInit callback
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID,
		chanCap, counterparty, version)
}

// OnChanOpenTry implements the IBCMiddleware interface
// If the channel is not fee enabled the underlying application version will be returned
// If the channel is fee enabled we merge the underlying application version with the ics29 version
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	// call underlying app's OnChanOpenTry callback
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	// call underlying app's OnChanOpenAck callback
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanOpenConfirm callback.
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanCloseInit callback.
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanCloseConfirm callback.
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCMiddleware interface.
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) exported.Acknowledgement {
	// first do the underlying ibc app callback
	ack := im.app.OnRecvPacket(ctx, packet, relayer)

	// postprocess the ibc application by executing a callback to the receiver smart contract
	// if the receiver of the transfer packet is a smart contract
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err == nil {
		// check if data.Receiver is a smart contract address
		// if it is a smart contract address
		// then call the smart-contract defined callback for RecvPacket and pass in the transfer packet data
		// if the callback returns an error, then return an error acknowledgement
        if isSmartContract(data.Receiver) {
            err := data.Receiver.recvPacketCallback(data)
            if err != nil {
                return channeltypes.NewErrorAcknowledgement(err)
            }
        }
	}
	return ack
}

// OnAcknowledgementPacket implements the IBCMiddleware interface
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// call underlying callback
	err := im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)

	// postprocess the ibc application by executing a callback to the sender smart contract
	// if the sender of the transfer packet is a smart contract
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err == nil {
		// check if data.Sender is a smart contract address
		// if it is a smart contract address
		// then call the smart-contract defined callback for RecvPacket and pass in the transfer packet data
		// if the callback returns an error, then return an error
        if isSmartContract(data.Sender) {
            data.Sender.ackPacketCallback(data)
        }
	}
	return err
}

// OnTimeoutPacket implements the IBCMiddleware interface
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	// call underlying callback
	err := im.app.OnTimeoutPacket(ctx, packet, relayer)

	// postprocess the ibc application by executing a callback to the sender smart contract
	// if the sender of the transfer packet is a smart contract
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err == nil {
		// check if data.Sender is a smart contract address
		// if it is a smart contract address
		// then call the smart-contract defined callback for RecvPacket and pass in the transfer packet data
		// if the callback returns an error, then return an error
        if isSmartContract(data.Sender) {
            data.Sender.timeoutPacketCallback(data)
        }
	}
	return err
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
	return im.chanWrapper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
	ack exported.Acknowledgement,
) error {
	return im.chanWrapper.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

// GetAppVersion returns the application version of the underlying application
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.chanWrapper.GetAppVersion(ctx, portID, channelID)
}
```