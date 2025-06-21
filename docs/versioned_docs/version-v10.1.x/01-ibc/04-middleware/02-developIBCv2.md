---
title: Create and integrate IBC v2 middleware
sidebar_label: Create and integrate IBC v2 middleware
sidebar_position: 2
slug: /ibc/middleware/developIBCv2
---

# Quick Navigation

1. [Create a custom IBC v2 middleware](#create-a-custom-ibc-v2-middleware)
2. [Implement `IBCModule` interface](#implement-ibcmodule-interface)
3. [WriteAckWrapper](#writeackwrapper)
4. [Integrate IBC v2 Middleware](#integrate-ibc-v2-middleware)
5. [Security Model](#security-model)
6. [Design Principles](#design-principles)

## Create a custom IBC v2 middleware

IBC middleware will wrap over an underlying IBC application (a base application or downstream middleware) and sits between core IBC and the base application.

:::warning
middleware developers must use the same serialization and deserialization method as in ibc-go's codec: transfertypes.ModuleCdc.[Must]MarshalJSON
:::

For middleware builders this means:

```go
import transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
transfertypes.ModuleCdc.[Must]MarshalJSON
func MarshalAsIBCDoes(ack channeltypes.Acknowledgement) ([]byte, error) {
	return transfertypes.ModuleCdc.MarshalJSON(&ack)
}
```

The interfaces a middleware must implement are found in [core/api](https://github.com/cosmos/ibc-go/blob/main/modules/core/api/module.go#L11). Note that this interface has changed from IBC classic. 

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

The `IBCModule` interface consists of the packet callbacks where custom logic is performed. 

### Packet callbacks

The packet callbacks are where the middleware performs most of its custom logic. The middleware may read the packet flow data and perform some additional packet handling, or it may modify the incoming data before it reaches the underlying application. This enables a wide degree of usecases, as a simple base application like token-transfer can be transformed for a variety of usecases by combining it with custom middleware, for example acting as a filter for which tokens can be sent and received.

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
	// Middleware may choose to do custom preprocessing logic before calling the underlying app OnRecvPacket
    // Middleware may choose to error early and return a RecvPacketResult Failure
    // Middleware may choose to modify the payload before passing on to OnRecvPacket though this
    // should only be done to support very advanced custom behavior
    // Middleware MUST NOT modify client identifiers and sequence
    doCustomPreProcessLogic()
    
	// call underlying app OnRecvPacket
    recvResult := im.app.OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
	if recvResult.Status == PACKET_STATUS_FAILURE {
		return recvResult
	}

    doCustomPostProcessLogic(recvResult) // middleware may modify recvResult
    
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
	// preprocessing logic may modify the acknowledgement before passing to 
	// the underlying app though this should only be done in advanced cases
	// Middleware may return error early
	// it MUST NOT change the identifiers of the clients or the sequence
	doCustomPreProcessLogic(payload, acknowledgement)

	// call underlying app OnAcknowledgementPacket
	err = im.app.OnAcknowledgementPacket(
		sourceClient, destinationClient, sequence,
		acknowledgement, payload, relayer
	)
	if err != nil {
		return err
	}

	// may perform some post acknowledgement logic and return error here
	return doCustomPostProcessLogic()
}
```

See [here](https://github.com/cosmos/ibc-go/blob/main/modules/apps/callbacks/v2/ibc_middleware.go#L236-L302) an example implementation of this callback for the Callbacks Middleware module.

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
	// Middleware may choose to do custom preprocessing logic before calling the underlying app OnTimeoutPacket
	// Middleware may return error early
	doCustomPreProcessLogic(payload)

	// call underlying app OnTimeoutPacket
	err = im.app.OnTimeoutPacket(
		sourceClient, destinationClient, sequence,
		payload, relayer
	)
	if err != nil {
		return err
	}

	// may perform some post timeout logic and return error here
	return doCustomPostProcessLogic()
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
// WriteAcknowledgement facilitates acknowledgment being written asynchronously
// The call stack flows from the IBC application to the IBC core handler
// Thus this function is called by the IBC app or a lower-level middleware
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	clientID string,
	sequence uint64,
	ack channeltypesv2.Acknowledgement,
) error {
	doCustomPreProcessLogic() // may modify acknowledgement

	return im.writeAckWrapper.WriteAcknowledgement(
		ctx, clientId, sequence, ack,
	)
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

## Security Model

IBC Middleware completely wraps all communication between IBC core and the application that it is wired with. Thus, the IBC Middleware has complete control to modify any packets and acknowledgements the underlying application receives or sends. Thus, if a chain chooses to wrap an application with a given middleware, that middleware is **completely trusted** and part of the application's security model. **Do not use middlewares that are untrusted.**

## Design Principles

The middleware follows a decorator pattern that wraps an underlying application's connection to the IBC core handlers. Thus, when implementing a middleware for a specific purpose, it is recommended to be as **unintrusive** as possible in the middleware design while still accomplishing the intended behavior.

The least intrusive middleware is stateless. They simply read the ICS26 callback arguments before calling the underlying app's callback and error if the arguments are not acceptable (e.g. whitelisting packets). Stateful middleware that are used solely for erroring are also very simple to build, an example of this would be a rate-limiting middleware that prevents transfer outflows from getting too high within a certain time frame.

Middleware that directly interfere with the payload or acknowledgement before passing control to the underlying app are way more intrusive to the underlying app processing. This makes such middleware more error-prone when implementing as incorrect handling can cause the underlying app to break or worse execute unexpected behavior. Moreover, such middleware typically needs to be built for a specific underlying app rather than being generic. An example of this is the packet-forwarding middleware which modifies the payload and is specifically built for transfer.

Middleware that modifies the payload or acknowledgement such that it is no longer readable by the underlying application is the most complicated middleware. Since it is not readable by the underlying apps, if these middleware write additional state into payloads and acknowledgements that get committed to IBC core provable state, there MUST be an equivalent counterparty middleware that is able to parse and interpret this additional state while also converting the payload and acknowledgment back to a readable form for the underlying application on its side. Thus, such middleware requires deployment on both sides of an IBC connection or the packet processing will break. This is the hardest type of middleware to implement, integrate and deploy. Thus, it is not recommended unless absolutely necessary to fulfill the given use case.
