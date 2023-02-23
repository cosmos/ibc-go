<!--
order: 2
-->

# Integration

Learn how to integrate IBC to your application and send data packets to other chains. {synopsis}

This document outlines the required steps to integrate and configure the [IBC
module](https://github.com/cosmos/ibc-go/tree/main/modules/core) to your Cosmos SDK application and
send fungible token transfers to other chains.

## Integrating the IBC module

Integrating the IBC module to your SDK-based application is straighforward. The general changes can be summarized in the following steps:

- Add required modules to the `module.BasicManager`
- Define additional `Keeper` fields for the new modules on the `App` type
- Add the module's `StoreKey`s and initialize their `Keeper`s
- Set up corresponding routers and routes for the `ibc` module
- Add the modules to the module `Manager`
- Add modules to `Begin/EndBlockers` and `InitGenesis`
- Update the module `SimulationManager` to enable simulations

### Module `BasicManager` and `ModuleAccount` permissions

The first step is to add the following modules to the `BasicManager`: `x/capability`, `x/ibc`,
and `x/ibc-transfer`. After that, we need to grant `Minter` and `Burner` permissions to
the `ibc-transfer` `ModuleAccount` to mint and burn relayed tokens.

### Integrating light clients

> Note that from v7 onwards, all light clients have to be explicitly registered in a chain's app.go and follow the steps listed below.
  This is in contrast to earlier versions of ibc-go when `07-tendermint` and `06-solomachine` were added out of the box.

All light clients must be registered with `module.BasicManager` in a chain's app.go file.

The following code example shows how to register the existing `ibctm.AppModuleBasic{}` light client implementation.

```diff
import (
  ...
+ ibctm "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint"
  ...
)

// app.go
var (
  ModuleBasics = module.NewBasicManager(
    // ...
    capability.AppModuleBasic{},
    ibc.AppModuleBasic{},
    transfer.AppModuleBasic{}, // i.e ibc-transfer module

    // register light clients on IBC
+   ibctm.AppModuleBasic{},
  )

  // module account permissions
  maccPerms = map[string][]string{
    // other module accounts permissions
    // ...
    ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
  }
)
```

### Application fields

Then, we need to register the `Keepers` as follows:

```go
// app.go
type App struct {
  // baseapp, keys and subspaces definitions

  // other keepers
  // ...
  IBCKeeper        *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
  TransferKeeper   ibctransferkeeper.Keeper // for cross-chain fungible token transfers

  // make scoped keepers public for test purposes
  ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
  ScopedTransferKeeper capabilitykeeper.ScopedKeeper

  /// ...
  /// module and simulation manager definitions
}
```

### Configure the `Keepers`

During initialization, besides initializing the IBC `Keepers` (for the `x/ibc`, and
`x/ibc-transfer` modules), we need to grant specific capabilities through the capability module
`ScopedKeepers` so that we can authenticate the object-capability permissions for each of the IBC
channels.

```go
func NewApp(...args) *App {
  // define codecs and baseapp

  // add capability keeper and ScopeToModule for ibc module
  app.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])

  // grant capabilities for the ibc and ibc-transfer modules
  scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
  scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

  // ... other modules keepers

  // Create IBC Keeper
  app.IBCKeeper = ibckeeper.NewKeeper(
    appCodec, keys[ibcexported.StoreKey], app.GetSubspace(ibcexported.ModuleName), app.StakingKeeper, app.UpgradeKeeper, scopedIBCKeeper,
  )

  // Create Transfer Keepers
  app.TransferKeeper = ibctransferkeeper.NewKeeper(
    appCodec, keys[ibctransfertypes.StoreKey], app.GetSubspace(ibctransfertypes.ModuleName),
    app.IBCKeeper.ChannelKeeper, app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
    app.AccountKeeper, app.BankKeeper, scopedTransferKeeper,
  )
  transferModule := transfer.NewAppModule(app.TransferKeeper)

  // .. continues
}
```

### Register `Routers`

IBC needs to know which module is bound to which port so that it can route packets to the
appropriate module and call the appropriate callbacks. The port to module name mapping is handled by
IBC's port `Keeper`. However, the mapping from module name to the relevant callbacks is accomplished
by the port
[`Router`](https://github.com/cosmos/ibc-go/blob/main/modules/core/05-port/types/router.go) on the
IBC module.

Adding the module routes allows the IBC handler to call the appropriate callback when processing a
channel handshake or a packet.

Currently, a `Router` is static so it must be initialized and set correctly on app initialization.
Once the `Router` has been set, no new routes can be added.

```go
// app.go
func NewApp(...args) *App {
  // .. continuation from above

  // Create static IBC router, add ibc-tranfer module route, then set and seal it
  ibcRouter := port.NewRouter()
  ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferModule)
  // Setting Router will finalize all routes by sealing router
  // No more routes can be added
  app.IBCKeeper.SetRouter(ibcRouter)

  // .. continues
```

### Module Managers

In order to use IBC, we need to add the new modules to the module `Manager` and to the `SimulationManager` in case your application supports [simulations](https://github.com/cosmos/cosmos-sdk/blob/main/docs/docs/building-modules/13-simulator.md).

```go
// app.go
func NewApp(...args) *App {
  // .. continuation from above

  app.mm = module.NewManager(
    // other modules
    // ...
    capability.NewAppModule(appCodec, *app.CapabilityKeeper),
    ibc.NewAppModule(app.IBCKeeper),
    transferModule,
  )

  // ...

  app.sm = module.NewSimulationManager(
    // other modules
    // ...
    capability.NewAppModule(appCodec, *app.CapabilityKeeper),
    ibc.NewAppModule(app.IBCKeeper),
    transferModule,
  )

  // .. continues
```

### Application ABCI Ordering

One addition from IBC is the concept of `HistoricalEntries` which are stored on the staking module.
Each entry contains the historical information for the `Header` and `ValidatorSet` of this chain which is stored
at each height during the `BeginBlock` call. The historical info is required to introspect the
past historical info at any given height in order to verify the light client `ConsensusState` during the
connection handhake.

```go
// app.go
func NewApp(...args) *App {
  // .. continuation from above

  // add staking and ibc modules to BeginBlockers
  app.mm.SetOrderBeginBlockers(
    // other modules ...
    stakingtypes.ModuleName, ibcexported.ModuleName,
  )

  // ...

  // NOTE: Capability module must occur first so that it can initialize any capabilities
  // so that other modules that want to create or claim capabilities afterwards in InitChain
  // can do so safely.
  app.mm.SetOrderInitGenesis(
    capabilitytypes.ModuleName,
    // other modules ...
    ibcexported.ModuleName, ibctransfertypes.ModuleName,
  )

  // .. continues
```

::: warning
**IMPORTANT**: The capability module **must** be declared first in `SetOrderInitGenesis`
:::

That's it! You have now wired up the IBC module and are now able to send fungible tokens across
different chains. If you want to have a broader view of the changes take a look into the SDK's
[`SimApp`](https://github.com/cosmos/ibc-go/blob/main/testing/simapp/app.go).

## Extending the client keeper

IBC-Go allows to extend the client keeper for supporting some extra use cases. Specifically,
the IBC keeper is defined as follows

```go
// Keeper defines each ICS keeper for IBC
type Keeper struct {
	// implements gRPC QueryServer interface
	types.QueryServer

	cdc codec.BinaryCodec

	ClientKeeper     clientexported.ClientKeeper
	ConnectionKeeper connectionkeeper.Keeper
	ChannelKeeper    channelkeeper.Keeper
	PortKeeper       portkeeper.Keeper
	Router           *porttypes.Router
}
```

Here `clientexported.ClientKeeper` is an interface rather a concrete keeper struct. One can 
instantiate `Keeper` with the original client keeper implementation under `02-client`, or an
extended version of it.

The extended keeper can support extra functionalities on top of the original client keeper, 
such that other modules or external services can learn light clients' internal state:

- adding new hooks and events upon certain state changes
- adding new queries and messages

As a concrete usage, one can implement a new hook that is triggered upon a new IBC header with a 
valid quorum certificate (i.e., a set of signatures from validators with >2/3 voting power).
This allows a Cosmos zone to timestamp such headers from another Cosmos zone, enabling use
cases for raising the economic security of Cosmos zones, e.g., Bitcoin timestamping and
mesh security.

One can implement an extended client keeper as follows

```go
type ExtendedKeeper struct {
	Keeper
}
```

Here `Keeper` is the original client keeper. Golang's composition feature makes this `ExtendedKeeper` 
to inherits all functionalities of `Keeper`, while allowing `ExtendedKeeper` to add new functions.

One can then replace the original client keeper object with the extended one in the IBC keeper in `app.go`
as follows

```go
	// initialize the extended client keeper
	extendedClientKeeper := ibcclientkeeper.ExtendedKeeper {
    Keeper: // ... original client keeper
    // ... fields of the Extended keeper
  }
  // replace the client keeper with the extended one in IBC keeper
	ibcKeeper.ClientKeeper = extendedClientKeeper
	// set IBC keeper for the app
	app.IBCKeeper = ibcKeeper
```

## Next {hide}

Learn about how to create [custom IBC modules](./apps/apps.md) for your application {hide}
