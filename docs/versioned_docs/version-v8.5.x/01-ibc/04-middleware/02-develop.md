---
title: Create a custom IBC middleware
sidebar_label: Create a custom IBC middleware
sidebar_position: 2
slug: /ibc/middleware/develop
---


# Create a custom IBC middleware

IBC middleware will wrap over an underlying IBC application (a base application or downstream middleware) and sits between core IBC and the base application.

The interfaces a middleware must implement are found [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/core/05-port/types/module.go).

```go
// Middleware implements the ICS26 Module interface
type Middleware interface {
  IBCModule // middleware has access to an underlying application which may be wrapped by more middleware
  ICS4Wrapper // middleware has access to ICS4Wrapper which may be core IBC Channel Handler or a higher-level middleware that wraps this middleware.
}
```

An `IBCMiddleware` struct implementing the `Middleware` interface, can be defined with its constructor as follows:

```go
// @ x/module_name/ibc_middleware.go

// IBCMiddleware implements the ICS26 callbacks and ICS4Wrapper for the fee middleware given the
// fee keeper and the underlying application.
type IBCMiddleware struct {
  app    porttypes.IBCModule
  keeper keeper.Keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application
func NewIBCMiddleware(app porttypes.IBCModule, k keeper.Keeper) IBCMiddleware {
  return IBCMiddleware{
    app:    app,
    keeper: k,
  }
}
```

## Implement `IBCModule` interface

`IBCMiddleware` is a struct that implements the [ICS-26 `IBCModule` interface (`porttypes.IBCModule`)](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/core/05-port/types/module.go#L14-L107). It is recommended to separate these callbacks into a separate file `ibc_middleware.go`.

> Note how this is analogous to implementing the same interfaces for IBC applications that act as base applications.

As will be mentioned in the [integration section](03-integration.md), this struct should be different than the struct that implements `AppModule` in case the middleware maintains its own internal state and processes separate SDK messages.

The middleware must have access to the underlying application, and be called before it during all ICS-26 callbacks. It may execute custom logic during these callbacks, and then call the underlying application's callback.

> Middleware **may** choose not to call the underlying application's callback at all. Though these should generally be limited to error cases.

The `IBCModule` interface consists of the channel handshake callbacks and packet callbacks. Most of the custom logic will be performed in the packet callbacks, in the case of the channel handshake callbacks, introducing the middleware requires consideration to the version negotiation and passing of capabilities.

### Channel handshake callbacks

#### Version negotiation

In the case where the IBC middleware expects to speak to a compatible IBC middleware on the counterparty chain, they must use the channel handshake to negotiate the middleware version without interfering in the version negotiation of the underlying application.

Middleware accomplishes this by formatting the version in a JSON-encoded string containing the middleware version and the application version. The application version may as well be a JSON-encoded string, possibly including further middleware and app versions, if the application stack consists of multiple milddlewares wrapping a base application. The format of the version is specified in ICS-30 as the following:

```json
{
  "<middleware_version_key>": "<middleware_version_value>",
  "app_version": "<application_version_value>"
}
```

The `<middleware_version_key>` key in the JSON struct should be replaced by the actual name of the key for the corresponding middleware (e.g. `fee_version`).

During the handshake callbacks, the middleware can unmarshal the version string and retrieve the middleware and application versions. It can do its negotiation logic on `<middleware_version_value>`, and pass the `<application_version_value>` to the underlying application.

> **NOTE**: Middleware that does not need to negotiate with a counterparty middleware on the remote stack will not implement the version unmarshalling and negotiation, and will simply perform its own custom logic on the callbacks without relying on the counterparty behaving similarly.

#### `OnChanOpenInit`

```go
func (im IBCMiddleware) OnChanOpenInit(
  ctx sdk.Context,
  order channeltypes.Order,
  connectionHops []string,
  portID string,
  channelID string,
  channelCap *capabilitytypes.Capability,
  counterparty channeltypes.Counterparty,
  version string,
) (string, error) {
  if version != "" {
    // try to unmarshal JSON-encoded version string and pass
    // the app-specific version to app callback.
    // otherwise, pass version directly to app callback.
    metadata, err := Unmarshal(version)
    if err != nil {
      // Since it is valid for fee version to not be specified,
      // the above middleware version may be for another middleware.
      // Pass the entire version string onto the underlying application.
      return im.app.OnChanOpenInit(
        ctx,
        order,
        connectionHops,
        portID,
        channelID,
        channelCap,
        counterparty,
        version,
      )
    }
    else {
      metadata = {
        // set middleware version to default value
        MiddlewareVersion: defaultMiddlewareVersion,
        // allow application to return its default version
        AppVersion: "",
      }
    }
  }

  doCustomLogic()

  // if the version string is empty, OnChanOpenInit is expected to return
  // a default version string representing the version(s) it supports
  appVersion, err := im.app.OnChanOpenInit(
    ctx,
    order,
    connectionHops,
    portID,
    channelID,
    channelCap,
    counterparty,
    metadata.AppVersion, // note we only pass app version here
  )
  if err != nil {
    return "", err
  }

  version := constructVersion(metadata.MiddlewareVersion, appVersion)

  return version, nil
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/ibc_middleware.go#L36-L83) an example implementation of this callback for the ICS-29 Fee Middleware module.

#### `OnChanOpenTry`

```go
func (im IBCMiddleware) OnChanOpenTry(
  ctx sdk.Context,
  order channeltypes.Order,
  connectionHops []string,
  portID,
  channelID string,
  channelCap *capabilitytypes.Capability,
  counterparty channeltypes.Counterparty,
  counterpartyVersion string,
) (string, error) {
  // try to unmarshal JSON-encoded version string and pass
  // the app-specific version to app callback.
  // otherwise, pass version directly to app callback.
  cpMetadata, err := Unmarshal(counterpartyVersion)
  if err != nil {
    return app.OnChanOpenTry(
      ctx,
      order,
      connectionHops,
      portID,
      channelID,
      channelCap,
      counterparty,
      counterpartyVersion,
    )
  }

  doCustomLogic()

  // Call the underlying application's OnChanOpenTry callback.
  // The try callback must select the final app-specific version string and return it.
  appVersion, err := app.OnChanOpenTry(
    ctx,
    order,
    connectionHops,
    portID,
    channelID,
    channelCap,
    counterparty,
    cpMetadata.AppVersion, // note we only pass counterparty app version here
  )
  if err != nil {
    return "", err
  }

  // negotiate final middleware version
  middlewareVersion := negotiateMiddlewareVersion(cpMetadata.MiddlewareVersion)
  version := constructVersion(middlewareVersion, appVersion)

  return version, nil
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/ibc_middleware.go#L88-L125) an example implementation of this callback for the ICS-29 Fee Middleware module.

#### `OnChanOpenAck`

```go
func (im IBCMiddleware) OnChanOpenAck(
  ctx sdk.Context,
  portID,
  channelID string,
  counterpartyChannelID string,
  counterpartyVersion string,
) error {
  // try to unmarshal JSON-encoded version string and pass
  // the app-specific version to app callback.
  // otherwise, pass version directly to app callback.
  cpMetadata, err = UnmarshalJSON(counterpartyVersion)
  if err != nil {
    return app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
  }

  if !isCompatible(cpMetadata.MiddlewareVersion) {
    return error
  }
  doCustomLogic()

  // call the underlying application's OnChanOpenTry callback
  return app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, cpMetadata.AppVersion)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/ibc_middleware.go#L128-L153)) an example implementation of this callback for the ICS-29 Fee Middleware module.

#### `OnChanOpenConfirm`

```go
func OnChanOpenConfirm(
  ctx sdk.Context,
  portID,
  channelID string,
) error {
  doCustomLogic()

  return app.OnChanOpenConfirm(ctx, portID, channelID)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/ibc_middleware.go#L156-L163) an example implementation of this callback for the ICS-29 Fee Middleware module.

#### `OnChanCloseInit`

```go
func OnChanCloseInit(
  ctx sdk.Context,
  portID,
  channelID string,
) error {
  doCustomLogic()

  return app.OnChanCloseInit(ctx, portID, channelID)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/ibc_middleware.go#L166-L188) an example implementation of this callback for the ICS-29 Fee Middleware module.

#### `OnChanCloseConfirm`

```go
func OnChanCloseConfirm(
  ctx sdk.Context,
  portID,
  channelID string,
) error {
  doCustomLogic()

  return app.OnChanCloseConfirm(ctx, portID, channelID)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/ibc_middleware.go#L191-L213) an example implementation of this callback for the ICS-29 Fee Middleware module.

#### Capabilities

The middleware should simply pass the capability in the callback arguments along to the underlying application so that it may be claimed by the base application. The base application will then pass the capability up the stack in order to authenticate an outgoing packet/acknowledgement, which you can check in the [`ICS4Wrapper` section](#ics-04-wrappers).

In the case where the middleware wishes to send a packet or acknowledgment without the involvement of the underlying application, it should be given access to the same `scopedKeeper` as the base application so that it can retrieve the capabilities by itself.

### Packet callbacks

The packet callbacks just like the handshake callbacks wrap the application's packet callbacks. The packet callbacks are where the middleware performs most of its custom logic. The middleware may read the packet flow data and perform some additional packet handling, or it may modify the incoming data before it reaches the underlying application. This enables a wide degree of usecases, as a simple base application like token-transfer can be transformed for a variety of usecases by combining it with custom middleware.

#### `OnRecvPacket`

```go
func (im IBCMiddleware) OnRecvPacket(
  ctx sdk.Context,
  packet channeltypes.Packet,
  relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
  doCustomLogic(packet)

  ack := app.OnRecvPacket(ctx, packet, relayer)

  doCustomLogic(ack) // middleware may modify outgoing ack
    
  return ack
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/ibc_middleware.go#L217-L238) an example implementation of this callback for the ICS-29 Fee Middleware module.

#### `OnAcknowledgementPacket`

```go
func (im IBCMiddleware) OnAcknowledgementPacket(
  ctx sdk.Context,
  packet channeltypes.Packet,
  acknowledgement []byte,
  relayer sdk.AccAddress,
) error {
  doCustomLogic(packet, ack)

  return app.OnAcknowledgementPacket(ctx, packet, ack, relayer)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/ibc_middleware.go#L242-L293) an example implementation of this callback for the ICS-29 Fee Middleware module.

#### `OnTimeoutPacket`

```go
func (im IBCMiddleware) OnTimeoutPacket(
  ctx sdk.Context,
  packet channeltypes.Packet,
  relayer sdk.AccAddress,
) error {
  doCustomLogic(packet)

  return app.OnTimeoutPacket(ctx, packet, relayer)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/ibc_middleware.go#L297-L335) an example implementation of this callback for the ICS-29 Fee Middleware module.

## ICS-04 wrappers

Middleware must also wrap ICS-04 so that any communication from the application to the `channelKeeper` goes through the middleware first. Similar to the packet callbacks, the middleware may modify outgoing acknowledgements and packets in any way it wishes.

To ensure optimal generalisability, the `ICS4Wrapper` abstraction serves to abstract away whether a middleware is the topmost middleware (and thus directly calling into the ICS-04 `channelKeeper`) or itself being wrapped by another middleware.

Remember that middleware can be stateful or stateless. When defining the stateful middleware's keeper, the `ics4Wrapper` field is included. Then the appropriate keeper can be passed when instantiating the middleware's keeper in `app.go`

```go
type Keeper struct {
  storeKey storetypes.StoreKey
  cdc      codec.BinaryCodec

  ics4Wrapper   porttypes.ICS4Wrapper
  channelKeeper types.ChannelKeeper
  portKeeper    types.PortKeeper
  ...
}
```

For stateless middleware, the `ics4Wrapper` can be passed on directly without having to instantiate a keeper struct for the middleware.

[The interface](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/core/05-port/types/module.go#L110-L133) looks as follows:

```go
// This is implemented by ICS4 and all middleware that are wrapping base application.
// The base application will call `sendPacket` or `writeAcknowledgement` of the middleware directly above them
// which will call the next middleware until it reaches the core IBC handler.
type ICS4Wrapper interface {
  SendPacket(
    ctx sdk.Context,
    chanCap *capabilitytypes.Capability,
    sourcePort string,
    sourceChannel string,
    timeoutHeight clienttypes.Height,
    timeoutTimestamp uint64,
    data []byte,
  ) (sequence uint64, err error)

  WriteAcknowledgement(
    ctx sdk.Context,
    chanCap *capabilitytypes.Capability,
    packet exported.PacketI,
    ack exported.Acknowledgement,
  ) error

  GetAppVersion(
    ctx sdk.Context,
    portID,
    channelID string,
  ) (string, bool)
}
```

:warning: In the following paragraphs, the methods are presented in pseudo code which has been kept general, not stating whether the middleware is stateful or stateless. Remember that when the middleware is stateful, `ics4Wrapper` can be accessed through the keeper.

Check out the references provided for an actual implementation to clarify, where the `ics4Wrapper` methods in `ibc_middleware.go` simply call the equivalent keeper methods where the actual logic resides.

### `SendPacket`

```go
func SendPacket(
  ctx sdk.Context,
  chanCap *capabilitytypes.Capability,
  sourcePort string,
  sourceChannel string,
  timeoutHeight clienttypes.Height,
  timeoutTimestamp uint64,
  appData []byte,
) (uint64, error) {
  // middleware may modify data
  data = doCustomLogic(appData)

  return ics4Wrapper.SendPacket(
    ctx, 
    chanCap, 
    sourcePort, 
    sourceChannel, 
    timeoutHeight, 
    timeoutTimestamp, 
    data,
  )
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/keeper/relay.go#L17-L27) an example implementation of this function for the ICS-29 Fee Middleware module.

### `WriteAcknowledgement`

```go
// only called for async acks
func WriteAcknowledgement(
  ctx sdk.Context,
  chanCap *capabilitytypes.Capability,
  packet exported.PacketI,
  ack exported.Acknowledgement,
) error {
  // middleware may modify acknowledgement
  ack_bytes = doCustomLogic(ack)

  return ics4Wrapper.WriteAcknowledgement(packet, ack_bytes)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/keeper/relay.go#L31-L55) an example implementation of this function for the ICS-29 Fee Middleware module.

### `GetAppVersion`

```go
// middleware must return the underlying application version
func GetAppVersion(
  ctx sdk.Context,
  portID,
  channelID string,
) (string, bool) {
  version, found := ics4Wrapper.GetAppVersion(ctx, portID, channelID)
  if !found {
    return "", false
  }

  if !MiddlewareEnabled {
    return version, true
  }

  // unwrap channel version
  metadata, err := Unmarshal(version)
  if err != nil {
    panic(fmt.Errof("unable to unmarshal version: %w", err))
  }

  return metadata.AppVersion, true
}
```

See [here](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/29-fee/keeper/relay.go#L58-L74) an example implementation of this function for the ICS-29 Fee Middleware module.
