---
title: Integration
sidebar_label: Integration
sidebar_position: 2
slug: /ibc/integration
---

# Integration

:::note Synopsis
Learn how to integrate IBC to your application
:::

This document outlines the required steps to integrate and configure the [IBC
module](https://github.com/cosmos/ibc-go/tree/main/modules/core) to your Cosmos SDK application and enable sending fungible token transfers to other chains. An [example app using ibc-go v10 is linked](https://github.com/gjermundgaraba/probe/tree/ibc/v10).

## Integrating the IBC module

Integrating the IBC module to your SDK-based application is straightforward. The general changes can be summarized in the following steps:

- [Define additional `Keeper` fields for the new modules on the `App` type](#add-application-fields-to-app).
- [Add the module's `StoreKey`s and initialize their `Keeper`s](#configure-the-keepers).
- [Create Application Stacks with Middleware](#create-application-stacks-with-middleware)
- [Set up IBC router and add route for the `transfer` module](#register-module-routes-in-the-ibc-router).
- [Grant permissions to `transfer`'s `ModuleAccount`](#module-account-permissions).
- [Add the modules to the module `Manager`](#module-manager-and-simulationmanager).
- [Update the module `SimulationManager` to enable simulations](#module-manager-and-simulationmanager).
- [Integrate light client modules (e.g. `07-tendermint`)](#integrating-light-clients).
- [Add modules to `Begin/EndBlockers` and `InitGenesis`](#application-abci-ordering).

### Add application fields to `App`

We need to register the core `ibc` and `transfer` `Keeper`s. To support the use of IBC v2, `transferv2` and `callbacksv2` must also be registered as follows:

```go title="app.go"
import (
  // other imports
  // ...
  ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
  ibctransferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
  // ibc v2 imports
  transferv2 "github.com/cosmos/ibc-go/v10/modules/apps/transfer/v2"
  ibccallbacksv2 "github.com/cosmos/ibc-go/v10/modules/apps/callbacks/v2"
)

type App struct {
  // baseapp, keys and subspaces definitions

  // other keepers
  // ...
  IBCKeeper        *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
  TransferKeeper   ibctransferkeeper.Keeper // for cross-chain fungible token transfers

  // ...
  // module and simulation manager definitions
}
```

### Configure the `Keeper`s

Initialize the IBC `Keeper`s (for core `ibc` and `transfer` modules), and any additional modules you want to include. 

:::note Notice
The capability module has been removed in ibc-go v10, therefore the `ScopedKeeper` has also been removed
:::

```go
import (
  // other imports
  // ...
  authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

  ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
  ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
  "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
  ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
  ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

func NewApp(...args) *App {
  // define codecs and baseapp

  // ... other module keepers

  // Create IBC Keeper
  app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibcexported.StoreKey]),
		app.GetSubspace(ibcexported.ModuleName),
		app.UpgradeKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

  // Create Transfer Keeper
  app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibctransfertypes.StoreKey]),
		app.GetSubspace(ibctransfertypes.ModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.MsgServiceRouter(),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

  // ... continues
}
```

### Create Application Stacks with Middleware

Middleware stacks in IBC allow you to wrap an `IBCModule` with additional logic for packets and acknowledgements. This is a chain of handlers that execute in order. The transfer stack below shows how to wire up transfer to use packet forward middleware, and the callbacks middleware. Note that the order is important. 

```go
// Create Transfer Stack for IBC Classic
maxCallbackGas := uint64(10_000_000)
wasmStackIBCHandler := wasm.NewIBCHandler(app.WasmKeeper, app.IBCKeeper.ChannelKeeper, app.IBCKeeper.ChannelKeeper)

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
  0, // retries on timeout
  packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
)
```

#### IBC v2 Application Stack

For IBC v2, an example transfer stack is shown below. In this case the transfer stack is using the callbacks middleware.

```go
// Create IBC v2 transfer middleware stack
// the callbacks gas limit is recommended to be 10M for use with wasm contracts
maxCallbackGas := uint64(10_000_000)
wasmStackIBCHandler := wasm.NewIBCHandler(app.WasmKeeper, app.IBCKeeper.ChannelKeeper, app.IBCKeeper.ChannelKeeper)

var ibcv2TransferStack ibcapi.IBCModule
	ibcv2TransferStack = transferv2.NewIBCModule(app.TransferKeeper)
	ibcv2TransferStack = ibccallbacksv2.NewIBCMiddleware(transferv2.NewIBCModule(app.TransferKeeper), app.IBCKeeper.ChannelKeeperV2, wasmStackIBCHandler, app.IBCKeeper.ChannelKeeperV2, maxCallbackGas)
```

### Register module routes in the IBC `Router`

IBC needs to know which module is bound to which port so that it can route packets to the
appropriate module and call the appropriate callbacks. The port to module name mapping is handled by
IBC's port `Keeper`. However, the mapping from module name to the relevant callbacks is accomplished
by the port
[`Router`](https://github.com/cosmos/ibc-go/blob/main/modules/core/05-port/types/router.go) on the
`ibc` module.

Adding the module routes allows the IBC handler to call the appropriate callback when processing a channel handshake or a packet.

Currently, a `Router` is static so it must be initialized and set correctly on app initialization.
Once the `Router` has been set, no new routes can be added.

```go title="app.go"
import (
  // other imports
  // ...
  porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types" 
  ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

func NewApp(...args) *App {
  // .. continuation from above

  // Create static IBC router, add transfer module route, then set and seal it
  ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
  // Setting Router will finalize all routes by sealing router
  // No more routes can be added
  app.IBCKeeper.SetRouter(ibcRouter)

  // ... continues
```

#### IBC v2 Router

With IBC v2, there is a new [router](https://github.com/cosmos/ibc-go/blob/main/modules/core/api/router.go) that needs to register the routes for a portID to a given IBCModule. 

```go
// IBC v2 router creation
	ibcRouterV2 := ibcapi.NewRouter()
	ibcRouterV2.AddRoute(ibctransfertypes.PortID, ibcv2TransferStack)
  // Setting Router will finalize all routes by sealing router
  // No more routes can be added
	app.IBCKeeper.SetRouterV2(ibcRouterV2)
```

### Module `Manager` and `SimulationManager`

In order to use IBC, we need to add the new modules to the module `Manager` and to the `SimulationManager`, in case your application supports [simulations](https://github.com/cosmos/cosmos-sdk/blob/main/docs/build/building-modules/14-simulator.md).

```go title="app.go"
import (
  // other imports
  // ...
  "github.com/cosmos/cosmos-sdk/types/module"

  ibc "github.com/cosmos/ibc-go/v10/modules/core"
  "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
)

func NewApp(...args) *App {
  // ... continuation from above

  app.ModuleManager = module.NewManager(
    // other modules
    // ...
    // highlight-start
+   ibc.NewAppModule(app.IBCKeeper),
+   transfer.NewAppModule(app.TransferKeeper),
    // highlight-end
  )

  // ...

  app.simulationManager = module.NewSimulationManagerFromAppModules(
    // other modules
    // ...
    app.ModuleManager.Modules,
    map[string]module.AppModuleSimulation{},
  )

  // ... continues
```

### Module account permissions

After that, we need to grant `Minter` and `Burner` permissions to
the `transfer` `ModuleAccount` to mint and burn relayed tokens.

```go title="app.go"
import (
  // other imports
  // ...
  "github.com/cosmos/cosmos-sdk/types/module"
  authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

  // highlight-next-line
+ ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

// app.go
var (
  // module account permissions
  maccPerms = map[string][]string{
    // other module accounts permissions
    // ...
    ibctransfertypes.ModuleName: {authtypes.Minter, authtypes.Burner},
  }
)
```

### Integrating light clients

> Note that from v10 onwards, all light clients are expected to implement the [`LightClientInterface` interface](../03-light-clients/01-developer-guide/02-light-client-module.md#implementing-the-lightclientmodule-interface) defined by core IBC, and have to be explicitly registered in a chain's app.go. This is in contrast to earlier versions of ibc-go when `07-tendermint` and `06-solomachine` were added out of the box. Follow the steps below to integrate the `07-tendermint` light client. 

All light clients must be registered with `module.Manager` in a chain's app.go file. The following code example shows how to instantiate `07-tendermint` light client module and register its `ibctm.AppModule`. 

```go title="app.go"
import (
  // other imports
  // ...
  "github.com/cosmos/cosmos-sdk/types/module"
  // highlight-next-line
+ ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

// app.go
// after sealing the IBC router
clientKeeper := app.IBCKeeper.ClientKeeper
storeProvider := app.IBCKeeper.ClientKeeper.GetStoreProvider()

tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
clientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)
// ...
app.ModuleManager = module.NewManager(
  // ...
  ibc.NewAppModule(app.IBCKeeper),
  transfer.NewAppModule(app.TransferKeeper), // i.e ibc-transfer module

  // register light clients on IBC
  // highlight-next-line
+ ibctm.NewAppModule(tmLightClientModule),
)
```

#### Allowed Clients Params

The allowed clients parameter defines an allow list of client types supported by the chain. The 
default value is a single-element list containing the [`AllowedClients`](https://github.com/cosmos/ibc-go/blob/main/modules/core/02-client/types/client.pb.go#L248-L253) wildcard (`"*"`). Alternatively, the parameter
may be set with a list of client types (e.g. `"06-solomachine","07-tendermint","09-localhost"`).
A client type that is not registered on this list will fail upon creation or on genesis validation.
Note that, since the client type is an arbitrary string, chains must not register two light clients
which return the same value for the `ClientType()` function, otherwise the allow list check can be
bypassed.

### Application ABCI ordering

One addition from IBC is the concept of `HistoricalInfo` which is stored in the Cosmos SDK `x/staking` module. The number of records stored by `x/staking` is controlled by the `HistoricalEntries` parameter which stores `HistoricalInfo` on a per-height basis.
Each entry contains the historical information for the `Header` and `ValidatorSet` of this chain which is stored
at each height during the `BeginBlock` call. The `HistoricalInfo` is required to introspect a blockchain's prior state at a given height in order to verify the light client `ConsensusState` during the
connection handshake. 

```go title="app.go"
import (
  // other imports
  // ...
  stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
  ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
  ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
  ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

func NewApp(...args) *App {
  // ... continuation from above

  // add x/staking, ibc and transfer modules to BeginBlockers
  app.ModuleManager.SetOrderBeginBlockers(
    // other modules ...
    stakingtypes.ModuleName,
    ibcexported.ModuleName,
    ibctransfertypes.ModuleName,
  )
  app.ModuleManager.SetOrderEndBlockers(
    // other modules ...
    stakingtypes.ModuleName,
    ibcexported.ModuleName,
    ibctransfertypes.ModuleName,
  )

  // ...

  genesisModuleOrder := []string{
    // other modules
    // ...
    ibcexported.ModuleName,
    ibctransfertypes.ModuleName,
  }
  app.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)

  // ... continues
```

That's it! You have now wired up the IBC module and the `transfer` module, and are now able to send fungible tokens across
different chains. If you want to have a broader view of the changes take a look into the SDK's
[`SimApp`](https://github.com/cosmos/ibc-go/blob/main/testing/simapp/app.go).
