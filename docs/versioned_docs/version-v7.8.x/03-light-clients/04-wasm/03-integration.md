---
title: Integration
sidebar_label: Integration
sidebar_position: 3
slug: /ibc/light-clients/wasm/integration
---

# Integration

Learn how to integrate the `08-wasm` module in a chain binary and about the recommended approaches depending on whether the [`x/wasm` module](https://github.com/CosmWasm/wasmd/tree/main/x/wasm) is already used in the chain. The following document only applies for Cosmos SDK chains. 

## Importing the `08-wasm` module

`08-wasm` has no stable releases yet. To use it, you need to import the git commit that contains the module with the compatible versions of `ibc-go` and `wasmvm`. To do so, run the following command with the desired git commit in your project:

```sh
go get github.com/cosmos/ibc-go/modules/light-clients/08-wasm@7ee2a2452b79d0bc8316dc622a1243afa058e8cb
```

You can find the version matrix in [here](../../../../docs/03-light-clients/04-wasm/03-integration.md#importing-the-08-wasm-module).

## `app.go` setup

The sample code below shows the relevant integration points in `app.go` required to setup the `08-wasm` module in a chain binary. Since `08-wasm` is a light client module itself, please check out as well the section [Integrating light clients](../../01-ibc/02-integration.md#integrating-light-clients) for more information:

```go
// app.go
import (
  ...
  "github.com/cosmos/cosmos-sdk/runtime"
  
  cmtos "github.com/cometbft/cometbft/libs/os"

  ibcwasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm"
  ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
  ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
  ...
)

...

// Register the AppModule for the 08-wasm module
ModuleBasics = module.NewBasicManager(
  ...
  ibcwasm.AppModuleBasic{},
  ...
)

// Add 08-wasm Keeper
type SimApp struct {
  ...
  WasmClientKeeper ibcwasmkeeper.Keeper
  ...
}

func NewSimApp(
  logger log.Logger,
  db dbm.DB,
  traceStore io.Writer,
  loadLatest bool,
  appOpts servertypes.AppOptions,
  baseAppOptions ...func(*baseapp.BaseApp),
) *SimApp {
  ...
  keys := sdk.NewKVStoreKeys(
    ...
    ibcwasmtypes.StoreKey,
  )

  // Instantiate 08-wasm's keeper
  // This sample code uses a constructor function that
  // accepts a pointer to an existing instance of Wasm VM.
  // This is the recommended approach when the chain
  // also uses `x/wasm`, and then the Wasm VM instance
  // can be shared.
  app.WasmClientKeeper = ibcwasmkeeper.NewKeeperWithVM(
    appCodec,
    keys[wasmtypes.StoreKey],
    app.IBCKeeper.ClientKeeper,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
    wasmVM,
    app.GRPCQueryRouter(),
  )  
  app.ModuleManager = module.NewManager(
    // SDK app modules
    ...
    ibcwasm.NewAppModule(app.WasmClientKeeper),
  ) 
  app.ModuleManager.SetOrderBeginBlockers(
    ...
    ibcwasmtypes.ModuleName,
    ...
  ) 
  app.ModuleManager.SetOrderEndBlockers(
    ...
    ibcwasmtypes.ModuleName,
    ...
  ) 
  genesisModuleOrder := []string{
    ...
    ibcwasmtypes.ModuleName,
    ...
  }
  app.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)
  app.ModuleManager.SetOrderExportGenesis(genesisModuleOrder...)
  ...

	// initialize BaseApp
  app.SetInitChainer(app.InitChainer)
  ...

  // must be before Loading version
  if manager := app.SnapshotManager(); manager != nil {
    err := manager.RegisterExtensions(
      ibcwasmkeeper.NewWasmSnapshotter(app.CommitMultiStore(), &app.WasmClientKeeper),
    )
    if err != nil {
      panic(fmt.Errorf("failed to register snapshot extension: %s", err))
    }
  }
  ...

  if loadLatest {
    ...

    ctx := app.BaseApp.NewUncachedContext(true, cmtproto.Header{})

    // Initialize pinned codes in wasmvm as they are not persisted there
    if err := ibcwasmkeeper.InitializePinnedCodes(ctx, app.appCodec); err != nil {
      cmtos.Exit(fmt.Sprintf("failed initialize pinned codes %s", err))
    }
  }
}
```

## Keeper instantiation

When it comes to instantiating `08-wasm`'s keeper there are two recommended ways of doing it. Choosing one or the other will depend on whether the chain already integrates [`x/wasm`](https://github.com/CosmWasm/wasmd/tree/main/x/wasm) or not.

### If `x/wasm` is present

If the chain where the module is integrated uses `x/wasm` then we recommend that both `08-wasm` and `x/wasm` share the same Wasm VM instance. Having two separate Wasm VM instances is still possible, but care should be taken to make sure that both instances do not share the directory when the VM stores blobs and various caches, otherwise unexpected behaviour is likely to happen.

In order to share the Wasm VM instance please follow the guideline below. Please note that this requires `x/wasm`v0.41 or above.

- Instantiate the Wasm VM in `app.go` with the parameters of your choice.
- [Create an `Option` with this Wasm VM instance](https://github.com/CosmWasm/wasmd/blob/db93d7b6c7bb6f4a340d74b96a02cec885729b59/x/wasm/keeper/options.go#L21-L25).
- Add the option created in the previous step to a slice and [pass it to the `x/wasm NewKeeper` constructor function](https://github.com/CosmWasm/wasmd/blob/db93d7b6c7bb6f4a340d74b96a02cec885729b59/x/wasm/keeper/keeper_cgo.go#L36).
- Pass the pointer to the Wasm VM instance to `08-wasm` [`NewKeeperWithVM` constructor function](https://github.com/cosmos/ibc-go/blob/b306e7a706e1f84a5e11af0540987bd68de9bae5/modules/light-clients/08-wasm/keeper/keeper.go#L38-L46).

The code to set this up would look something like this:

```go
// app.go
import (
  ...
  "github.com/cosmos/cosmos-sdk/runtime"
  
  wasmvm "github.com/CosmWasm/wasmvm"
  wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
  wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
  
  ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
  ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
  ...
)

...

// instantiate the Wasm VM with the chosen parameters
wasmer, err := wasmvm.NewVM(
  dataDir, 
  availableCapabilities, 
  contractMemoryLimit,
  contractDebugMode, 
  memoryCacheSize,
)
if err != nil {
  panic(err)
}

// create an Option slice (or append to an existing one)
// with the option to use a custom Wasm VM instance
wasmOpts = []wasmkeeper.Option{
  wasmkeeper.WithWasmEngine(wasmer),
}

// the keeper will use the provided Wasm VM instance,
// instead of instantiating a new one
app.WasmKeeper = wasmkeeper.NewKeeper(
  appCodec,
  keys[wasmtypes.StoreKey],
  app.AccountKeeper,
  app.BankKeeper,
  app.StakingKeeper,
  distrkeeper.NewQuerier(app.DistrKeeper),
  app.IBCFeeKeeper, // ISC4 Wrapper: fee IBC middleware
  app.IBCKeeper.ChannelKeeper,
  &app.IBCKeeper.PortKeeper,
  scopedWasmKeeper,
  app.TransferKeeper,
  app.MsgServiceRouter(),
  app.GRPCQueryRouter(),
  wasmDir,
  wasmConfig,
  availableCapabilities,
  authtypes.NewModuleAddress(govtypes.ModuleName).String(),
  wasmOpts...,
)

app.WasmClientKeeper = ibcwasmkeeper.NewKeeperWithVM(
  appCodec,
  keys[ibcwasmtypes.StoreKey],
  app.IBCKeeper.ClientKeeper,
  authtypes.NewModuleAddress(govtypes.ModuleName).String(),
  wasmer, // pass the Wasm VM instance to `08-wasm` keeper constructor
  app.GRPCQueryRouter(),
)
...
```

### If `x/wasm` is not present

If the chain does not use [`x/wasm`](https://github.com/CosmWasm/wasmd/tree/main/x/wasm), even though it is still possible to use the method above from the previous section
(e.g. instantiating a Wasm VM in app.go an pass it to 08-wasm's [`NewKeeperWithVM` constructor function](https://github.com/cosmos/ibc-go/blob/b306e7a706e1f84a5e11af0540987bd68de9bae5/modules/light-clients/08-wasm/keeper/keeper.go#L38-L46), since there would be no need in this case to share the Wasm VM instance with another module, you can use the [`NewKeeperWithConfig` constructor function](https://github.com/cosmos/ibc-go/blob/b306e7a706e1f84a5e11af0540987bd68de9bae5/modules/light-clients/08-wasm/keeper/keeper.go#L82-L90) and provide the Wasm VM configuration parameters of your choice instead. A Wasm VM instance will be created in `NewKeeperWithConfig`. The parameters that can set are:

- `DataDir` is the [directory for Wasm blobs and various caches](https://github.com/CosmWasm/wasmvm/blob/1638725b25d799f078d053391945399cb35664b1/lib.go#L25). In `wasmd` this is set to the [`wasm` folder under the home directory](https://github.com/CosmWasm/wasmd/blob/36416def20effe47fb77f29f5ba35a003970fdba/app/app.go#L578).
- `SupportedCapabilities` is a comma separated [list of capabilities supported by the chain](https://github.com/CosmWasm/wasmvm/blob/1638725b25d799f078d053391945399cb35664b1/lib.go#L26). [`wasmd` sets this to all the available capabilities](https://github.com/CosmWasm/wasmd/blob/36416def20effe47fb77f29f5ba35a003970fdba/app/app.go#L586), but 08-wasm only requires `iterator`.
- `MemoryCacheSize` sets [the size in MiB of an in-memory cache for e.g. module caching](https://github.com/CosmWasm/wasmvm/blob/1638725b25d799f078d053391945399cb35664b1/lib.go#L29C16-L29C104). It is not consensus-critical and should be defined on a per-node basis, often in the range 100 to 1000 MB. [`wasmd` reads this value of](https://github.com/CosmWasm/wasmd/blob/36416def20effe47fb77f29f5ba35a003970fdba/app/app.go#L579). Default value is 256.
- `ContractDebugMode` is a [flag to enable/disable printing debug logs from the contract to STDOUT](https://github.com/CosmWasm/wasmvm/blob/1638725b25d799f078d053391945399cb35664b1/lib.go#L28). This should be false in production environments. Default value is false.

Another configuration parameter of the Wasm VM is the contract memory limit (in MiB), which is [set to 32](https://github.com/cosmos/ibc-go/blob/b306e7a706e1f84a5e11af0540987bd68de9bae5/modules/light-clients/08-wasm/types/config.go#L8), [following the example of `wasmd`](https://github.com/CosmWasm/wasmd/blob/36416def20effe47fb77f29f5ba35a003970fdba/x/wasm/keeper/keeper.go#L32-L34). This parameter is not configurable by users of `08-wasm`.

The following sample code shows how the keeper would be constructed using this method:

```go
// app.go
import (
  ...
  "github.com/cosmos/cosmos-sdk/runtime"

  ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
  ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
  ...
)

...

// homePath is the path to the directory where the data
// directory for Wasm blobs and caches will be created
wasmConfig := ibcwasmtypes.WasmConfig{
  DataDir:               filepath.Join(homePath, "ibc_08-wasm_client_data"),
  SupportedCapabilities: "iterator",
  ContractDebugMode:     false,
}
app.WasmClientKeeper = ibcwasmkeeper.NewKeeperWithConfig(
  appCodec,
  keys[ibcwasmtypes.StoreKey],
  app.IBCKeeper.ClientKeeper, 
  authtypes.NewModuleAddress(govtypes.ModuleName).String(),
  wasmConfig,
  app.GRPCQueryRouter(),
)
```

Check out also the [`WasmConfig` type definition](https://github.com/cosmos/ibc-go/blob/b306e7a706e1f84a5e11af0540987bd68de9bae5/modules/light-clients/08-wasm/types/config.go#L21-L31) for more information on each of the configurable parameters. Some parameters allow node-level configurations. There is additionally the function [`DefaultWasmConfig`](https://github.com/cosmos/ibc-go/blob/b306e7a706e1f84a5e11af0540987bd68de9bae5/modules/light-clients/08-wasm/types/config.go#L36) available that returns a configuration with the default values.

### Options

The `08-wasm` module comes with an options API inspired by the one in `x/wasm`.
Currently the only option available is the `WithQueryPlugins` option, which allows registration of custom query plugins for the `08-wasm` module. The use of this API is optional and it is only required if the chain wants to register custom query plugins for the `08-wasm` module.

#### `WithQueryPlugins`

By default, the `08-wasm` module does not support any queries. However, it is possible to register custom query plugins for [`QueryRequest::Custom`](https://github.com/CosmWasm/cosmwasm/blob/v1.5.0/packages/std/src/query/mod.rs#L45) and [`QueryRequest::Stargate`](https://github.com/CosmWasm/cosmwasm/blob/v1.5.0/packages/std/src/query/mod.rs#L54-L61).

Assuming that the keeper is not yet instantiated, the following sample code shows how to register query plugins for the `08-wasm` module.

We first construct a [`QueryPlugins`](https://github.com/cosmos/ibc-go/blob/b306e7a706e1f84a5e11af0540987bd68de9bae5/modules/light-clients/08-wasm/types/querier.go#L78-L87) object with the desired query plugins:

```go
queryPlugins := ibcwasmtypes.QueryPlugins {
  Custom: MyCustomQueryPlugin(),
  // `myAcceptList` is a `[]string` containing the list of gRPC query paths that the chain wants to allow for the `08-wasm` module to query.
  // These queries must be registered in the chain's gRPC query router, be deterministic, and track their gas usage.
  // The `AcceptListStargateQuerier` function will return a query plugin that will only allow queries for the paths in the `myAcceptList`.
  // The query responses are encoded in protobuf unlike the implementation in `x/wasm`.
  Stargate: ibcwasmtypes.AcceptListStargateQuerier(myAcceptList),
}
```

You may leave any of the fields in the `QueryPlugins` object as `nil` if you do not want to register a query plugin for that query type.

Then, we pass the `QueryPlugins` object to the `WithQueryPlugins` option:

```go
querierOption := ibcwasmtypes.WithQueryPlugins(&queryPlugins)
```

Finally, we pass the option to the `NewKeeperWithConfig` or `NewKeeperWithVM` constructor function during [Keeper instantiation](#keeper-instantiation):

```diff
app.WasmClientKeeper = ibcwasmkeeper.NewKeeperWithConfig(
  appCodec,
  keys[ibcwasmtypes.StoreKey],
  app.IBCKeeper.ClientKeeper, 
  authtypes.NewModuleAddress(govtypes.ModuleName).String(),
  wasmConfig,
  app.GRPCQueryRouter(),
+ querierOption,
)
```

```diff
app.WasmClientKeeper = ibcwasmkeeper.NewKeeperWithVM(
  appCodec,
  keys[wasmtypes.StoreKey],
  app.IBCKeeper.ClientKeeper,
  authtypes.NewModuleAddress(govtypes.ModuleName).String(),
  wasmer, // pass the Wasm VM instance to `08-wasm` keeper constructor
  app.GRPCQueryRouter(),
+ querierOption,
)
```

## Updating `AllowedClients`

In order to use the `08-wasm` module chains must update the [`AllowedClients` parameter in the 02-client submodule](https://github.com/cosmos/ibc-go/blob/v7.3.0/proto/ibc/core/client/v1/client.proto#L104) of core IBC. This can be configured directly in the application upgrade handler with the sample code below:

```go
import (
  ...
  ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
  ...
)

...

func CreateWasmUpgradeHandler(
  mm *module.Manager,
  configurator module.Configurator,
  clientKeeper clientkeeper.Keeper,
) upgradetypes.UpgradeHandler {
  return func(goCtx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)
    // explicitly update the IBC 02-client params, adding the wasm client type
    params := clientKeeper.GetParams(ctx)
    params.AllowedClients = append(params.AllowedClients, ibcwasmtypes.Wasm)
    clientKeeper.SetParams(ctx, params)

    return mm.RunMigrations(goCtx, configurator, vm)
  }
}
```

Or alternatively the parameter can be updated via a governance proposal (see at the bottom of section [`Creating clients`](../01-developer-guide/09-setup.md#creating-clients) for an example of how to do this).

## Adding the module to the store

As part of the upgrade migration you must also add the module to the upgrades store.

```go
func (app SimApp) RegisterUpgradeHandlers() {

  ...

  if upgradeInfo.Name == UpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
    storeUpgrades := storetypes.StoreUpgrades{
      Added: []string{
        ibcwasmtypes.ModuleName,
      },
    }

    // configure store loader that checks if version == upgradeHeight and applies store upgrades
    app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
  }
}
```

## Adding snapshot support

In order to use the `08-wasm` module chains are required to register the `WasmSnapshotter` extension in the snapshot manager. This snapshotter takes care of persisting the external state, in the form of contract code, of the Wasm VM instance to disk when the chain is snapshotted. [This code](https://github.com/cosmos/ibc-go/blob/b306e7a706e1f84a5e11af0540987bd68de9bae5/modules/light-clients/08-wasm/testing/simapp/app.go#L747-L755) should be placed in `NewSimApp` function in `app.go`.

## Pin byte codes at start

Wasm byte codes should be pinned to the WasmVM cache on every application start, therefore [this code](https://github.com/cosmos/ibc-go/blob/b306e7a706e1f84a5e11af0540987bd68de9bae5/modules/light-clients/08-wasm/testing/simapp/app.go#L786-L791) should be placed in `NewSimApp` function in `app.go`.
