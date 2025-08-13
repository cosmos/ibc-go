---
title: Integration
sidebar_label: Integration
sidebar_position: 1
slug: /apps/packet-forward-middleware/integration
---

# Integration

This document provides instructions on integrating and configuring the Packet Forward Middleware (PFM) within your
existing chain implementation.
The integration steps include the following:

1. [Import the PFM, initialize the PFM Module & Keeper, initialize the store keys and module params, and initialize the Begin/End Block logic and InitGenesis order](#example-integration-of-the-packet-forward-middleware)
2. [Configure the IBC application stack including the transfer module](#configuring-the-transfer-application-stack-with-packet-forward-middleware)
3. [Configuration of additional options such as timeout period, number of retries on timeout, refund timeout period, and fee percentage](#configurable-options-in-the-packet-forward-middleware)

Integration of the PFM should take approximately 20 minutes.

## Example integration of the Packet Forward Middleware

```go
// app.go

// Import the packet forward middleware
import (
    "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward"
    packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/keeper"
    packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/types"
)

...

// Register the AppModule for the packet forward middleware module
ModuleBasics = module.NewBasicManager(
    ...
    packetforward.AppModuleBasic{},
    ...
)

...

// Add packet forward middleware Keeper
type App struct {
 ...
 PacketForwardKeeper *packetforwardkeeper.Keeper
 ...
}

...

// Create store keys
keys := sdk.NewKVStoreKeys(
    ...
    packetforwardtypes.StoreKey,
    ...
)

...

// Initialize the packet forward middleware Keeper
// It's important to note that the PFM Keeper must be initialized before the Transfer Keeper
app.PacketForwardKeeper = packetforwardkeeper.NewKeeper(
    appCodec,
    keys[packetforwardtypes.StoreKey],
    nil, // will be zero-value here, reference is set later on with SetTransferKeeper.
    app.IBCKeeper.ChannelKeeper,
    appKeepers.DistrKeeper,
    app.BankKeeper,
    app.IBCKeeper.ChannelKeeper,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)

// Initialize the transfer module Keeper
app.TransferKeeper = ibctransferkeeper.NewKeeper(
    appCodec,
    keys[ibctransfertypes.StoreKey],
    app.GetSubspace(ibctransfertypes.ModuleName),
    app.PacketForwardKeeper,
    app.IBCKeeper.ChannelKeeper,
    &app.IBCKeeper.PortKeeper,
    app.AccountKeeper,
    app.BankKeeper,
    scopedTransferKeeper,
)

app.PacketForwardKeeper.SetTransferKeeper(app.TransferKeeper)

// See the section below for configuring an application stack with the packet forward middleware

...

// Register packet forward middleware AppModule
app.moduleManager = module.NewManager(
    ...
    packetforward.NewAppModule(app.PacketForwardKeeper, app.GetSubspace(packetforwardtypes.ModuleName)),
)

...

// Add packet forward middleware to begin blocker logic
app.moduleManager.SetOrderBeginBlockers(
    ...
    packetforwardtypes.ModuleName,
    ...
)

// Add packet forward middleware to end blocker logic
app.moduleManager.SetOrderEndBlockers(
    ...
    packetforwardtypes.ModuleName,
    ...
)

// Add packet forward middleware to init genesis logic
app.moduleManager.SetOrderInitGenesis(
    ...
    packetforwardtypes.ModuleName,
    ...
)

// Add packet forward middleware to init params keeper
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
    ...
    paramsKeeper.Subspace(packetforwardtypes.ModuleName).WithKeyTable(packetforwardtypes.ParamKeyTable())
    ...
}
```

## Configuring the transfer application stack with Packet Forward Middleware

Here is an example of how to create an application stack using `transfer` and `packet-forward-middleware`.
The following `transferStack` is configured in `app/app.go` and added to the IBC `Router`.
The in-line comments describe the execution flow of packets between the application stack and IBC core.

For more information on configuring an IBC application stack see the ibc-go docs [here](https://github.com/cosmos/ibc-go/blob/e69a833de764fa0f5bdf0338d9452fd6e579a675/docs/docs/04-middleware/01-ics29-fee/02-integration.md#configuring-an-application-stack-with-fee-middleware).

```go
// Create Transfer Stack
// SendPacket, since it is originating from the application to core IBC:
// transferKeeper.SendPacket -> packetforward.SendPacket -> channel.SendPacket

// RecvPacket, message that originates from core IBC and goes down to app, the flow is the other way
// channel.RecvPacket -> packetforward.OnRecvPacket -> transfer.OnRecvPacket

// transfer stack contains (from top to bottom):
// - Packet Forward Middleware
// - Transfer
var transferStack ibcporttypes.IBCModule
transferStack = transfer.NewIBCModule(app.TransferKeeper)
transferStack = packetforward.NewIBCMiddleware(
    transferStack,
    app.PacketForwardKeeper,
    0, // retries on timeout
    packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp, // forward timeout
)

// Add transfer stack to IBC Router
ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
```

## Configurable options in the Packet Forward Middleware

The Packet Forward Middleware has several configurable options available when initializing the IBC application stack.
You can see these passed in as arguments to `packetforward.NewIBCMiddleware` and they include the number of retries that
will be performed on a forward timeout, the timeout period that will be used for a forward, and the timeout period that
will be used for performing refunds in the case that a forward is taking too long.

Additionally, there is a fee percentage parameter that can be set in `InitGenesis`, this is an optional parameter that
can be used to take a fee from each forwarded packet which will then be distributed to the community pool. In the
`OnRecvPacket` callback `ForwardTransferPacket` is invoked which will attempt to subtract a fee from the forwarded
packet amount if the fee percentage is non-zero.

- Retries On Timeout - how many times will a forward be re-attempted in the case of a timeout.
- Timeout Period - how long can a forward be in progress before giving up.
- Refund Timeout - how long can a forward be in progress before issuing a refund back to the original source chain.
- Fee Percentage - % of the forwarded packet amount which will be subtracted and distributed to the community pool.
