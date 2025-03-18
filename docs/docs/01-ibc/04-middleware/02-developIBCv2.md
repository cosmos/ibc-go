---
title: Create and integrate IBC v2 middleware
sidebar_label: Create and integrate IBC v2 middleware
sidebar_position: 2
slug: /ibc/middleware/developIBCv2
---


# Create a custom IBC v2 middleware

IBC middleware will wrap over an underlying IBC application (a base application or downstream middleware) and sits between core IBC and the base application.

The interfaces a middleware must implement are found in [core/api](https://github.com/cosmos/ibc-go/blob/main/modules/core/api/module.go#L11). Note that this interface has chanhged from IBC classic. 

An `IBCMiddleware` struct implementing the `Middleware` interface, can be defined with its constructor as follows:

```go
// @ x/module_name/ibc_middleware.go

// IBCMiddleware implements the IBCv2 middleware interface
type IBCMiddleware struct {
  app                   api.IBCModule // underlying app or middleware
  writeAckWrapper       api. WriteAcknowledgementWrapper // writes acknowledgement for an async acknowledgement
  PacketDataUnmarshaler api.PacketDataUnmarshaler // optional interface
  keeper                types.Keeper // required for stateful middleware
  // Keeper may include middleware specific keeper and the ChannelKeeperV2

  // additional middleware specific fields 
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application
func NewIBCMiddleware(app api.IBCModule, 
writeAckWrapper api.WriteAcknowledgementWrapper, 
k types.Keeper
) IBCMiddleware {
  return IBCMiddleware{
    app:             app,
    writeAckWrapper: writeAckWrapper,
    keeper:          k,
  }
}
```

:::note
The ICS4Wrapper has been removed in IBC v2 and there are no channel handshake callbacks, a writeAckWrapper has been added to the interface
:::

## Implement `IBCModule` interface

`IBCMiddleware` is a struct that implements the [`IBCModule` interface (`api.IBCModule`)](https://github.com/cosmos/ibc-go/blob/main/modules/core/api/module.go#L11-L53). It is recommended to separate these callbacks into a separate file `ibc_middleware.go`.

> Note how this is analogous to implementing the same interfaces for IBC applications that act as base applications.

The middleware must have access to the underlying application, and be called before it during all ICS-26 callbacks. It may execute custom logic during these callbacks, and then call the underlying application's callback.

> Middleware **may** choose not to call the underlying application's callback at all. Though these should generally be limited to error cases.

The `IBCModule` interface consists of the packet callbacks where cutom logic is performed. 

### Packet callbacks

The packet callbacks are where the middleware performs most of its custom logic. The middleware may read the packet flow data and perform some additional packet handling, or it may modify the incoming data before it reaches the underlying application. This enables a wide degree of usecases, as a simple base application like token-transfer can be transformed for a variety of usecases by combining it with custom middleware, for example acting as a filter for which tokens can be sent and recieved.

#### `OnRecvPacket`

```go
func (im IBCMiddleware) OnRecvPacket(
  ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) channeltypesv2.RecvPacketResult {
	recvResult := im.app.OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)

  doCustomLogic(recvResult) // middleware may modify success acknowledgment
    
  return recvResult
}
```

See [here](https://github.com/cosmos/ibc-go/blob/main/modules/apps/callbacks/v2/ibc_middleware.go#L161-L230) an example implementation of this callback for the Callbacks Middleware module.

#### `OnAcknowledgementPacket`

```go
func (im IBCMiddleware) OnAcknowledgementPacket(
  ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {

  doCustomLogic(payload, acknowledgement)

  return nil
}
```

See [here](hhttps://github.com/cosmos/ibc-go/blob/main/modules/apps/callbacks/v2/ibc_middleware.go#L236-L302) an example implementation of this callback for the Callbacks Middleware module.

#### `OnTimeoutPacket`

```go
func (im IBCMiddleware) OnTimeoutPacket(
  ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
  doCustomLogic(payload)

  return nil
}
```

See [here](https://github.com/cosmos/ibc-go/blob/main/modules/apps/callbacks/v2/ibc_middleware.go#L309-L367) an example implementation of this callback for the Callbacks Middleware module.

### WriteAckWrapper

Middleware must also wrap the `WriteAcknowledgement` interface so that any acknowledgement written by the application passes through the middleware first. This allows middleware to modify or delay writing an acknowledgment before committed to the IBC store. 

```go
// WithWriteAckWrapper sets the WriteAcknowledgementWrapper for the middleware.
func (im *IBCMiddleware) WithWriteAckWrapper(writeAckWrapper api.WriteAcknowledgementWrapper) {
	im.writeAckWrapper = writeAckWrapper
}

// GetWriteAckWrapper returns the WriteAckWrapper
func (im *IBCMiddleware) GetWriteAckWrapper() api.WriteAcknowledgementWrapper {
	return im.writeAckWrapper
}
```

### `WriteAcknowledgement`

This is where the middleware acknowledgement handling is finalised. An example is shown in the [callbacks middleware](https://github.com/cosmos/ibc-go/blob/main/modules/apps/callbacks/v2/ibc_middleware.go#L369-L454)

```go
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	clientID string,
	sequence uint64,
	ack channeltypesv2.Acknowledgement,
) error {
  // packet and payload handling and validation

  // custom middleware logic, for example callbacks. 

return nil
}
```

## Integrate IBC v2 Middleware

Middleware should be registered within the module manager in `app.go`.

The order of middleware **matters**, function calls from IBC to the application travel from top-level middleware to the bottom middleware and then to the application. Function calls from the application to IBC goes through the bottom middleware in order to the top middleware and then to core IBC handlers. Thus the same set of middleware put in different orders may produce different effects.

### Example Integration

The example integration is detailed for an IBC v2 stack using transfer and the callbacks middleware.

```go
// Middleware Stacks
// initialising callbacks middleware  
	maxCallbackGas := uint64(10_000_000)
	wasmStackIBCHandler := wasm.NewIBCHandler(app.WasmKeeper, app.IBCKeeper.ChannelKeeper, app.IBCKeeper.ChannelKeeper)

// Create the transferv2 stack with transfer and callbacks middleware
  var ibcv2TransferStack ibcapi.IBCModule
	ibcv2TransferStack = transferv2.NewIBCModule(app.TransferKeeper)
	ibcv2TransferStack = ibccallbacksv2.NewIBCMiddleware(transferv2.NewIBCModule(app.TransferKeeper), app.IBCKeeper.ChannelKeeperV2, wasmStackIBCHandler, app.IBCKeeper.ChannelKeeperV2, maxCallbackGas)

// Create static IBC v2 router, add app routes, then set and seal it
  ibcRouterV2 := ibcapi.NewRouter()
	ibcRouterV2.AddRoute(ibctransfertypes.PortID, ibcv2TransferStack)
	app.IBCKeeper.SetRouterV2(ibcRouterV2)
```
