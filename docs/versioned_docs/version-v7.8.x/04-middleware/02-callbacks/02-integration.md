---
title: Integration
sidebar_label: Integration
sidebar_position: 2
slug: /middleware/callbacks/integration
---

# Integration

Learn how to integrate the callbacks middleware with IBC applications. The following document is intended for developers building on top of the Cosmos SDK and only applies for Cosmos SDK chains. 

The callbacks middleware is a minimal and stateless implementation of the IBC middleware interface. It does not have a keeper, nor does it store any state. It simply routes IBC middleware messages to the appropriate callback function, which is implemented by the secondary application. Therefore, it doesn't need to be registered as a module, nor does it need to be added to the module manager. It only needs to be added to the IBC application stack.

## Pre-requisite Readings

- [IBC middleware development](../../01-ibc/04-middleware/01-develop.md)
- [IBC middleware integration](../../01-ibc/04-middleware/02-integration.md)

The callbacks middleware, as the name suggests, plays the role of an IBC middleware and as such must be configured by chain developers to route and handle IBC messages correctly.
For Cosmos SDK chains this setup is done via the `app/app.go` file, where modules are constructed and configured in order to bootstrap the blockchain application.

## Importing the callbacks middleware

The callbacks middleware has no stable releases yet. To use it, you need to import the git commit that contains the module with the compatible version of `ibc-go`. To do so, run the following command with the desired git commit in your project:

```sh
go get github.com/cosmos/ibc-go/modules/apps/callbacks@17cf1260a9cdc5292512acc9bcf6336ef0d917f4
```

You can find the version matrix in [here](../../../../docs/04-middleware/01-callbacks/02-integration.md#importing-the-callbacks-middleware).

## Configuring an application stack with the callbacks middleware

As mentioned in [IBC middleware development](../../01-ibc/04-middleware/01-develop.md) an application stack may be composed of many or no middlewares that nest a base application.
These layers form the complete set of application logic that enable developers to build composable and flexible IBC application stacks.
For example, an application stack may just be a single base application like `transfer`, however, the same application stack composed with `29-fee` and `callbacks` will nest the `transfer` base application twice by wrapping it with the Fee Middleware module and then callbacks middleware.

The callbacks middleware also **requires** a secondary application that will receive the callbacks to implement the [`ContractKeeper`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/apps/callbacks/types/expected_keepers.go#L11-L83). Since the wasm module does not yet support the callbacks middleware, we will use the `mockContractKeeper` module in the examples below. You should replace this with a module that implements `ContractKeeper`.

### Transfer

See below for an example of how to create an application stack using `transfer`, `29-fee`, and `callbacks`. Feel free to omit the `29-fee` middleware if you do not want to use it.
The following `transferStack` is configured in `app/app.go` and added to the IBC `Router`.
The in-line comments describe the execution flow of packets between the application stack and IBC core.

```go
// Create Transfer Stack
// SendPacket, since it is originating from the application to core IBC:
// transferKeeper.SendPacket -> callbacks.SendPacket -> feeKeeper.SendPacket -> channel.SendPacket

// RecvPacket, message that originates from core IBC and goes down to app, the flow is the other way
// channel.RecvPacket -> fee.OnRecvPacket -> callbacks.OnRecvPacket -> transfer.OnRecvPacket

// transfer stack contains (from top to bottom):
// - IBC Fee Middleware
// - IBC Callbacks Middleware
// - Transfer

// create IBC module from bottom to top of stack
var transferStack porttypes.IBCModule
transferStack = transfer.NewIBCModule(app.TransferKeeper)
transferStack = ibccallbacks.NewIBCMiddleware(transferStack, app.IBCFeeKeeper, app.MockContractKeeper, maxCallbackGas)
transferICS4Wrapper := transferStack.(porttypes.ICS4Wrapper)
transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)
// Since the callbacks middleware itself is an ics4wrapper, it needs to be passed to the transfer keeper
app.TransferKeeper.WithICS4Wrapper(transferICS4Wrapper)

// Add transfer stack to IBC Router
ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
```

:::warning
The usage of `WithICS4Wrapper` after `transferStack`'s configuration is critical! It allows the callbacks middleware to do `SendPacket` callbacks and asynchronous `ReceivePacket` callbacks. You must do this regardless of whether you are using the `29-fee` middleware or not.
:::

### Interchain Accounts Controller

```go
// Create Interchain Accounts Stack
// SendPacket, since it is originating from the application to core IBC:
// icaControllerKeeper.SendTx -> callbacks.SendPacket -> fee.SendPacket -> channel.SendPacket

// initialize ICA module with mock module as the authentication module on the controller side
var icaControllerStack porttypes.IBCModule
icaControllerStack = ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp("", scopedICAMockKeeper))
app.ICAAuthModule = icaControllerStack.(ibcmock.IBCModule)
icaControllerStack = icacontroller.NewIBCMiddleware(icaControllerStack, app.ICAControllerKeeper)
icaControllerStack = ibccallbacks.NewIBCMiddleware(icaControllerStack, app.IBCFeeKeeper, app.MockContractKeeper, maxCallbackGas)
icaICS4Wrapper := icaControllerStack.(porttypes.ICS4Wrapper)
icaControllerStack = ibcfee.NewIBCMiddleware(icaControllerStack, app.IBCFeeKeeper)
// Since the callbacks middleware itself is an ics4wrapper, it needs to be passed to the ica controller keeper
app.ICAControllerKeeper.WithICS4Wrapper(icaICS4Wrapper)

// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
// channel.RecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket

var icaHostStack porttypes.IBCModule
icaHostStack = icahost.NewIBCModule(app.ICAHostKeeper)
icaHostStack = ibcfee.NewIBCMiddleware(icaHostStack, app.IBCFeeKeeper)

// Add ICA host and controller to IBC router ibcRouter.
AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
AddRoute(icahosttypes.SubModuleName, icaHostStack).
```

:::warning
The usage of `WithICS4Wrapper` here is also critical!
:::
