---
title: Integration
sidebar_label: Integration
sidebar_position: 2
slug: /middleware/callbacks/integration
---

# Integration

Learn how to integrate the callbacks middleware with IBC applications. The following document is intended for developers building on top of the Cosmos SDK and only applies for Cosmos SDK chains. 

:::tip
An example integration for an IBC v2 transfer stack using the callbacks middleware can be found in the [ibc-go module integration](../../01-ibc/02-integration.md) section
:::

The callbacks middleware is a minimal and stateless implementation of the IBC middleware interface. It does not have a keeper, nor does it store any state. It simply routes IBC middleware messages to the appropriate callback function, which is implemented by the secondary application. Therefore, it doesn't need to be registered as a module, nor does it need to be added to the module manager. It only needs to be added to the IBC application stack.

## Pre-requisite Readings

- [IBC middleware development](../../01-ibc/04-middleware/02-develop.md)
- [IBC middleware integration](../../01-ibc/04-middleware/03-integration.md)

The callbacks middleware, as the name suggests, plays the role of an IBC middleware and as such must be configured by chain developers to route and handle IBC messages correctly.
For Cosmos SDK chains this setup is done via the `app/app.go` file, where modules are constructed and configured in order to bootstrap the blockchain application.

## Configuring an application stack with the callbacks middleware

As mentioned in [IBC middleware development](../../01-ibc/04-middleware/02-develop.md) an application stack may be composed of many or no middlewares that nest a base application.
These layers form the complete set of application logic that enable developers to build composable and flexible IBC application stacks.
For example, an application stack may just be a single base application like `transfer`, however, the same application stack composed with `packet-forward-middleware` and `callbacks` will nest the `transfer` base application twice by wrapping it with the callbacks module and then packet forward middleware.

The callbacks middleware also **requires** a secondary application that will receive the callbacks to implement the [`ContractKeeper`](https://github.com/cosmos/ibc-go/blob/main/modules/apps/callbacks/types/expected_keepers.go#L12-L100). The wasmd contract keeper has been implemented [here](https://github.com/CosmWasm/wasmd/tree/main/x/wasm/keeper) and is referenced as the `WasmKeeper`.

### Transfer

See below for an example of how to create an application stack using `transfer`, `packet-forward-middleware`, and `callbacks`. Feel free to omit the `packet-forward-middleware` if you do not want to use it.
The following `transferStack` is configured in `app/app.go` and added to the IBC `Router`.
The in-line comments describe the execution flow of packets between the application stack and IBC core.

```go
// Create Transfer Stack
// SendPacket, since it is originating from the application to core IBC:
// transferKeeper.SendPacket -> callbacks.SendPacket -> feeKeeper.SendPacket -> channel.SendPacket

// RecvPacket, message that originates from core IBC and goes down to app, the flow is the other way
// channel.RecvPacket -> fee.OnRecvPacket -> callbacks.OnRecvPacket -> transfer.OnRecvPacket

// transfer stack contains (from top to bottom):
// - IBC Packet Forward Middleware
// - IBC Callbacks Middleware
// - Transfer

// initialise the gas limit for callbacks, recommended to be 10M for use with cosmwasm contracts
maxCallbackGas := uint64(10_000_000)

// the keepers for the callbacks middleware
wasmStackIBCHandler := wasm.NewIBCHandler(app.WasmKeeper, app.IBCKeeper.ChannelKeeper, app.IBCKeeper.ChannelKeeper)

// create IBC module from bottom to top of stack
// Create Transfer Stack
	var transferStack porttypes.IBCModule
	transferStack = transfer.NewIBCModule(app.TransferKeeper)
// callbacks wraps the transfer stack as its base app, and uses PacketForwardKeeper as the ICS4Wrapper
// i.e. packet-forward-middleware is higher on the stack and sits between callbacks and the ibc channel keeper
// Since this is the lowest level middleware of the transfer stack, it should be the first entrypoint for transfer keeper's
// WriteAcknowledgement.
	cbStack := ibccallbacks.NewIBCMiddleware(transferStack, app.PacketForwardKeeper, wasmStackIBCHandler, maxCallbackGas)
	transferStack = packetforward.NewIBCMiddleware(
		cbStack,
		app.PacketForwardKeeper,
		0,
		packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
	)
	app.TransferKeeper.WithICS4Wrapper(cbStack)

// Create static IBC router, add app routes, then set and seal it
	ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
	ibcRouter.AddRoute(wasmtypes.ModuleName, wasmStackIBCHandler)
	app.IBCKeeper.SetRouter(ibcRouter)
```
