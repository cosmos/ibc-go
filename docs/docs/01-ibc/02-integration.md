---
title: Integration
sidebar_label: Integration
sidebar_position: 2
slug: /ibc/integration
---

# Integration

:::note Synopsis
Learn how to integrate IBC to your application and send data packets to other chains.
:::

This document outlines the required steps to integrate and configure the [IBC
module](https://github.com/cosmos/ibc-go/tree/main/modules/core) to your Cosmos SDK application and
send fungible token transfers to other chains.

## Integrating the IBC module

Integrating the IBC module to your SDK-based application is straightforward. The general changes can be summarized in the following steps:

- [Define additional `Keeper` fields for the new modules on the `App` type](#add-application-fields-to-app).
- [Add the module's `StoreKey`s and initialize their `Keeper`s](#configure-the-keepers).
- [Set up IBC router and add route for the `transfer` module](#register-module-routes-in-the-ibc-router).
- [Grant permissions to `transfer`'s `ModuleAccount`](#module-account-permissions).
- [Add the modules to the module `Manager`](#module-manager-and-simulationmanager).
- [Update the module `SimulationManager` to enable simulations](#module-manager-and-simulationmanager).
- [Integrate light client modules (e.g. `07-tendermint`)](#integrating-light-clients).
- [Add modules to `Begin/EndBlockers` and `InitGenesis`](#application-abci-ordering).

### Add application fields to `App`

We need to register the core `ibc` and `transfer` `Keeper`s as follows:

```go title="app.go"
import (
  // other imports
  // ...
  capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
  ibckeeper "github.com/cosmos/ibc-go/v9/modules/core/keeper"
  ibctransferkeeper "github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
)

type App struct {
  // baseapp, keys and subspaces definitions

  // other keepers
  // ...
  IBCKeeper        *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
  TransferKeeper   ibctransferkeeper.Keeper // for cross-chain fungible token transfers

  // make scoped keepers public for test purposes
  ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
  ScopedTransferKeeper capabilitykeeper.ScopedKeeper

  // ...
  // module and simulation manager definitions
}
```

### Configure the `Keeper`s

During initialization, besides initializing the IBC `Keeper`s (for core `ibc` and `transfer` modules), we need to grant specific capabilities through the capability module `ScopedKeeper`s so that we can authenticate the object-capability permissions for each of the IBC channels.

```go
import (
  // other imports
  // ...
  authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

  capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
  capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
  ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
  ibckeeper "github.com/cosmos/ibc-go/v9/modules/core/keeper"
  "github.com/cosmos/ibc-go/v9/modules/apps/transfer"
  ibctransfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
  ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
)

func NewApp(...args) *App {
  // define codecs and baseapp

  // add capability keeper and ScopeToModule for ibc module
  app.CapabilityKeeper = capabilitykeeper.NewKeeper(
    appCodec,
    keys[capabilitytypes.StoreKey],
    memKeys[capabilitytypes.MemStoreKey],
  )

  // grant capabilities for the ibc and transfer modules
  scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
  scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

  // ... other module keepers

  // Create IBC Keeper
  app.IBCKeeper = ibckeeper.NewKeeper(
    appCodec,
    keys[ibcexported.StoreKey],
    app.GetSubspace(ibcexported.ModuleName),
    app.UpgradeKeeper,
    scopedIBCKeeper,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
  )

  // Create Transfer Keeper
  app.TransferKeeper = ibctransferkeeper.NewKeeper(
    appCodec,
    keys[ibctransfertypes.StoreKey],
    app.GetSubspace(ibctransfertypes.ModuleName),
    app.IBCKeeper.ChannelKeeper,
    app.IBCKeeper.ChannelKeeper,
    app.IBCKeeper.PortKeeper,
    app.AccountKeeper,
    app.BankKeeper,
    scopedTransferKeeper,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
  )
  transferModule := transfer.NewIBCModule(app.TransferKeeper)

  // ... continues
}
```

### Register module routes in the IBC `Router`

IBC needs to know which module is bound to which port so that it can route packets to the
appropriate module and call the appropriate callbacks. The port to module name mapping is handled by
IBC's port `Keeper`. However, the mapping from module name to the relevant callbacks is accomplished
by the port
[`Router`](https://github.com/cosmos/ibc-go/blob/main/modules/core/05-port/types/router.go) on the
`ibc` module.

Adding the module routes allows the IBC handler to call the appropriate callback when processing a
channel handshake or a packet.

Currently, a `Router` is static so it must be initialized and set correctly on app initialization.
Once the `Router` has been set, no new routes can be added.

```go title="app.go"
import (
  // other imports
  // ...
  porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types" 
  ibctransfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
)

func NewApp(...args) *App {
  // .. continuation from above

  // Create static IBC router, add transfer module route, then set and seal it
  ibcRouter := porttypes.NewRouter()
  ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferModule)
  // Setting Router will finalize all routes by sealing router
  // No more routes can be added
  app.IBCKeeper.SetRouter(ibcRouter)

  // ... continues
```

### Module `Manager` and `SimulationManager`

In order to use IBC, we need to add the new modules to the module `Manager` and to the `SimulationManager`, in case your application supports [simulations](https://github.com/cosmos/cosmos-sdk/blob/main/docs/build/building-modules/14-simulator.md).

```go title="app.go"
import (
  // other imports
  // ...
  "github.com/cosmos/cosmos-sdk/types/module"

  ibc "github.com/cosmos/ibc-go/v9/modules/core"
  "github.com/cosmos/ibc-go/modules/capability"
  "github.com/cosmos/ibc-go/v9/modules/apps/transfer"
)

func NewApp(...args) *App {
  // ... continuation from above

  app.ModuleManager = module.NewManager(
    // other modules
    // ...
    // highlight-start
+   capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
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
+ ibctransfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
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

#### Integrating light clients

> Note that from v9 onwards, all light clients are expected to implement the [`LightClientInterface` interface](../03-light-clients/01-developer-guide/02-light-client-module.md#implementing-the-lightclientmodule-interface) defined by core IBC, and have to be explicitly registered in a chain's app.go. This is in contrast to earlier versions of ibc-go when `07-tendermint` and `06-solomachine` were added out of the box. Follow the steps below to integrate the `07-tendermint` light client.

All light clients must be registered with `module.Manager` in a chain's app.go file.

The following code example shows how to instantiate `07-tendermint` light client module and register its `ibctm.AppModule`.

```go title="app.go"
import (
  // other imports
  // ...
  "github.com/cosmos/cosmos-sdk/types/module"
  // highlight-next-line
+ ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
)

// app.go
// after sealing the IBC router

storeProvider := app.IBCKeeper.ClientKeeper.GetStoreProvider()

tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
app.IBCKeeper.ClientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)
app.ModuleManager = module.NewManager(
  // ...
  capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
  ibc.NewAppModule(app.IBCKeeper),
  transfer.NewAppModule(app.TransferKeeper), // i.e ibc-transfer module

  // register light clients on IBC
  // highlight-next-line
+ ibctm.NewAppModule(tmLightClientModule),
)
```

### Application ABCI ordering

One addition from IBC is the concept of `HistoricalInfo` which is stored in the Cosmos SDK `x/staking` module. The number of records stored by `x/staking` is controlled by the `HistoricalEntries` parameter which stores `HistoricalInfo` on a per height basis.
Each entry contains the historical information for the `Header` and `ValidatorSet` of this chain which is stored
at each height during the `BeginBlock` call. The `HistoricalInfo` is required to introspect a blockchain's prior state at a given height in order to verify the light client `ConsensusState` during the
connection handshake. 

```go title="app.go"
import (
  // other imports
  // ...
  stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
  capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
  ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
  ibckeeper "github.com/cosmos/ibc-go/v9/modules/core/keeper"
  ibctransfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
)

func NewApp(...args) *App {
  // ... continuation from above

  // add x/staking, ibc and transfer modules to BeginBlockers
  // capability module's Beginblocker must come before any
  // modules using capabilities (e.g. IBC)
  app.ModuleManager.SetOrderBeginBlockers(
    // other modules ...
    capabilitytypes.ModuleName,
    stakingtypes.ModuleName,
    ibcexported.ModuleName,
    ibctransfertypes.ModuleName,
  )
  app.ModuleManager.SetOrderEndBlockers(
    // other modules ...
    stakingtypes.ModuleName,
    ibcexported.ModuleName,
    ibctransfertypes.ModuleName,
    capabilitytypes.ModuleName,
  )

  // ...

  // NOTE: Capability module must occur first so that it can initialize any capabilities
  // so that other modules that want to create or claim capabilities afterwards in InitChain
  // can do so safely.
  genesisModuleOrder := []string{
    capabilitytypes.ModuleName,
    // other modules
    // ...
    ibcexported.ModuleName,
    ibctransfertypes.ModuleName,
  }
  app.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)

  // ... continues
```

:::warning
**IMPORTANT**: The capability module **must** be declared first in `SetOrderBeginBlockers` and `SetOrderInitGenesis`.
:::

That's it! You have now wired up the IBC module and the `transfer` module, and are now able to send fungible tokens across
different chains. If you want to have a broader view of the changes take a look into the SDK's
[`SimApp`](https://github.com/cosmos/ibc-go/blob/main/testing/simapp/app.go).
