<!--
order: 1
-->

# IBC middleware

Learn how to write your own custom middleware to wrap an IBC application, and understand how to hook different middleware to IBC base applications to form different IBC application stacks {synopsis}.

This document serves as a guide for middleware developers who want to write their own middleware and for chain developers who want to use IBC middleware on their chains.

IBC applications are designed to be self-contained modules that implement their own application-specific logic through a set of interfaces with the core IBC handlers. These core IBC handlers, in turn, are designed to enforce the correctness properties of IBC (transport, authentication, ordering) while delegating all application-specific handling to the IBC application modules. However, there are cases where some functionality may be desired by many applications, yet not appropriate to place in core IBC.

Middleware allows developers to define the extensions as separate modules that can wrap over the base application. This middleware can thus perform its own custom logic, and pass data into the application so that it may run its logic without being aware of the middleware's existence. This allows both the application and the middleware to implement its own isolated logic while still being able to run as part of a single packet flow.

## Pre-requisite readings

- [IBC Overview](../overview.md) {prereq}
- [IBC Integration](../integration.md) {prereq}
- [IBC Application Developer Guide](../apps.md) {prereq}

## Definitions

`Middleware`: A self-contained module that sits between core IBC and an underlying IBC application during packet execution. All messages between core IBC and underlying application must flow through middleware, which may perform its own custom logic.

`Underlying Application`: An underlying application is the application that is directly connected to the middleware in question. This underlying application may itself be middleware that is chained to a base application.

`Base Application`: A base application is an IBC application that does not contain any middleware. It may be nested by 0 or multiple middleware to form an application stack.

`Application Stack (or stack)`: A stack is the complete set of application logic (middleware(s) +  base application) that gets connected to core IBC. A stack may be just a base application, or it may be a series of middlewares that nest a base application.

## Create a custom IBC middleware

IBC middleware will wrap over an underlying IBC application and sits between core IBC and the application. It has complete control in modifying any message coming from IBC to the application, and any message coming from the application to core IBC. Thus, middleware must be completely trusted by chain developers who wish to integrate them, however this gives them complete flexibility in modifying the application(s) they wrap.

#### Interfaces

```go
// Middleware implements the ICS26 Module interface
type Middleware interface {
    porttypes.IBCModule // middleware has acccess to an underlying application which may be wrapped by more middleware
    ics4Wrapper: ICS4Wrapper // middleware has access to ICS4Wrapper which may be core IBC Channel Handler or a higher-level middleware that wraps this middleware.
}
```

```typescript
// This is implemented by ICS4 and all middleware that are wrapping base application.
// The base application will call `sendPacket` or `writeAcknowledgement` of the middleware directly above them
// which will call the next middleware until it reaches the core IBC handler.
type ICS4Wrapper interface {
    SendPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet exported.Packet) error
    WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet exported.Packet, ack exported.Acknowledgement) error
    GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool)
}
```

### Implement `IBCModule` interface and callbacks

The `IBCModule` is a struct that implements the [ICS-26 interface (`porttypes.IBCModule`)](https://github.com/cosmos/ibc-go/blob/main/modules/core/05-port/types/module.go#L11-L106). It is recommended to separate these callbacks into a separate file `ibc_module.go`. As will be mentioned in the [integration section](./integration.md), this struct should be different than the struct that implements `AppModule` in case the middleware maintains its own internal state and processes separate SDK messages.

The middleware must have access to the underlying application, and be called before during all ICS-26 callbacks. It may execute custom logic during these callbacks, and then call the underlying application's callback. Middleware **may** choose not to call the underlying application's callback at all. Though these should generally be limited to error cases.

In the case where the IBC middleware expects to speak to a compatible IBC middleware on the counterparty chain, they must use the channel handshake to negotiate the middleware version without interfering in the version negotiation of the underlying application.

Middleware accomplishes this by formatting the version in a JSON-encoded string containing the middleware version and the application version. The application version may as well be a JSON-encoded string, possibly including further middleware and app versions, if the application stack consists of multiple milddlewares wrapping a base application. The format of the version is specified in ICS-30 as the following:

```json
{"<middleware_version_key>":"<middleware_version_value>","app_version":"<application_version_value>"}
```

The `<middleware_version_key>` key in the JSON struct should be replaced by the actual name of the key for the corresponding middleware (e.g. `fee_version`).

During the handshake callbacks, the middleware can unmarshal the version string and retrieve the middleware and application versions. It can do its negotiation logic on `<middleware_version_value>`, and pass the `<application_version_value>` to the underlying application.

The middleware should simply pass the capability in the callback arguments along to the underlying application so that it may be claimed by the base application. The base application will then pass the capability up the stack in order to authenticate an outgoing packet/acknowledgement.

In the case where the middleware wishes to send a packet or acknowledgment without the involvement of the underlying application, it should be given access to the same `scopedKeeper` as the base application so that it can retrieve the capabilities by itself.

### Handshake callbacks

#### `OnChanOpenInit`

```go
func (im IBCModule) OnChanOpenInit(
    ctx sdk.Context,
    order channeltypes.Order,
    connectionHops []string,
    portID string,
    channelID string,
    channelCap *capabilitytypes.Capability,
    counterparty channeltypes.Counterparty,
    version string,
) (string, error) {
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

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L34-L82) an example implementation of this callback for the ICS29 Fee Middleware module.

#### `OnChanOpenTry`

```go
func OnChanOpenTry(
    ctx sdk.Context,
    order channeltypes.Order,
    connectionHops []string,
    portID,
    channelID string,
    channelCap *capabilitytypes.Capability,
    counterparty channeltypes.Counterparty,
    counterpartyVersion string,
) (string, error) {
    doCustomLogic()

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

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L84-L124) an example implementation of this callback for the ICS29 Fee Middleware module.

#### `OnChanOpenAck`

```go
func OnChanOpenAck(
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

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L126-L152) an example implementation of this callback for the ICS29 Fee Middleware module.

### `OnChanOpenConfirm`

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

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L154-L162) an example implementation of this callback for the ICS29 Fee Middleware module.

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

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L164-L187) an example implementation of this callback for the ICS29 Fee Middleware module.

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

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L189-L212) an example implementation of this callback for the ICS29 Fee Middleware module.

**NOTE**: Middleware that does not need to negotiate with a counterparty middleware on the remote stack will not implement the version unmarshalling and negotiation, and will simply perform its own custom logic on the callbacks without relying on the counterparty behaving similarly.

### Packet callbacks

The packet callbacks just like the handshake callbacks wrap the application's packet callbacks. The packet callbacks are where the middleware performs most of its custom logic. The middleware may read the packet flow data and perform some additional packet handling, or it may modify the incoming data before it reaches the underlying application. This enables a wide degree of usecases, as a simple base application like token-transfer can be transformed for a variety of usecases by combining it with custom middleware.

#### `OnRecvPacket`

```go
func OnRecvPacket(
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

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L214-L237) an example implementation of this callback for the ICS29 Fee Middleware module.

#### `OnAcknowledgementPacket`

```go
func OnAcknowledgementPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    acknowledgement []byte,
    relayer sdk.AccAddress,
) error {
    doCustomLogic(packet, ack)

    return app.OnAcknowledgementPacket(ctx, packet, ack, relayer)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L239-L292) an example implementation of this callback for the ICS29 Fee Middleware module.

#### `OnTimeoutPacket`

```go
func OnTimeoutPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    relayer sdk.AccAddress,
) error {
    doCustomLogic(packet)

    return app.OnTimeoutPacket(ctx, packet, relayer)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L294-L334) an example implementation of this callback for the ICS29 Fee Middleware module.

### ICS-4 wrappers

Middleware must also wrap ICS-4 so that any communication from the application to the `channelKeeper` goes through the middleware first. Similar to the packet callbacks, the middleware may modify outgoing acknowledgements and packets in any way it wishes.

#### `SendPacket`

```go
func SendPacket(
    ctx sdk.Context,
    chanCap *capabilitytypes.Capability,
    appPacket exported.PacketI,
) {
    // middleware may modify packet
    packet = doCustomLogic(appPacket)

    return ics4Keeper.SendPacket(ctx, chanCap, packet)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L336-L343) an example implementation of this function for the ICS29 Fee Middleware module.

#### `WriteAcknowledgement`

```go
// only called for async acks
func WriteAcknowledgement(
    ctx sdk.Context,
    chanCap *capabilitytypes.Capability,
    packet exported.PacketI,
    ack exported.Acknowledgement,
) {
    // middleware may modify acknowledgement
    ack_bytes = doCustomLogic(ack)

    return ics4Keeper.WriteAcknowledgement(packet, ack_bytes)
}
```

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L345-L353) an example implementation of this function for the ICS29 Fee Middleware module.

#### `GetAppVersion`

```go
// middleware must return the underlying application version 
func GetAppVersion(
    ctx sdk.Context,
    portID,
    channelID string,
) (string, bool) {
    version, found := ics4Keeper.GetAppVersion(ctx, portID, channelID)
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

See [here](https://github.com/cosmos/ibc-go/blob/48a6ae512b4ea42c29fdf6c6f5363f50645591a2/modules/apps/29-fee/ibc_middleware.go#L355-L358) an example implementation of this function for the ICS29 Fee Middleware module.
