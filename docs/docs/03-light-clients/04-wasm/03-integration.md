---
title: Integration
sidebar_label: Integration
sidebar_position: 3
slug: /ibc/light-clients/wasm/integration
---

# Integration

Learn how to integrate the `08-wasm` module in a chain binary and about the recommended approaches depending on whether the [`x/wasm` module](https://github.com/CosmWasm/wasmd/tree/main/x/wasm) is already used in the chain. The following document only applies for Cosmos SDK chains. {synopsis}

## `app.go` setup

The sample code below shows the relevant integration points in `app.go` required to setup the `08-wasm` module in a chain binary. Since `08-wasm` is a light client module itself, please check out as well the section [Integrating light clients](../../01-ibc/02-integration.md#integrating-light-clients) for more information:

```go
// app.go
import (
  ...
  "github.com/cosmos/cosmos-sdk/runtime"
  
  cmtos "github.com/cometbft/cometbft/libs/os"

  wasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm"
  wasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
  wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
  ...
)

...

// Register the AppModule for the 08-wasm module
ModuleBasics = module.NewBasicManager(
  ...
  wasm.AppModuleBasic{},
  ...
)

// Add 08-wasm Keeper
type SimApp struct {
  ...
  WasmClientKeeper wasmkeeper.Keeper
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
    wasmtypes.StoreKey,
  ) 

  // Instantiate 08-wasm's keeper
  // This sample code uses a constructor function that
  // accepts a pointer to an existing instance of Wasm VM.
  // This is the recommended approach when the chain
  // also uses `x/wasm`, and then the Wasm VM instance
  // can be shared.
  // This sample code uses also an implementation of the 
  // wasmvm.Querier interface (querier). If nil is passed
  // instead, then a default querier will be used that
  // returns an error for all query types.
  // See the section below for more information.
  app.WasmClientKeeper = wasmkeeper.NewKeeperWithVM(
    appCodec,
    runtime.NewKVStoreService(keys[wasmtypes.StoreKey]),
    app.IBCKeeper.ClientKeeper,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
    wasmVM,
    querier,
  )  
  app.ModuleManager = module.NewManager(
    // SDK app modules
    ...
    wasm.NewAppModule(app.WasmClientKeeper),
  ) 
  app.ModuleManager.SetOrderBeginBlockers(
    ...
    wasmtypes.ModuleName,
    ...
  ) 
  app.ModuleManager.SetOrderEndBlockers(
    ...
    wasmtypes.ModuleName,
    ...
  ) 
  genesisModuleOrder := []string{
    ...
    wasmtypes.ModuleName,
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
      wasmkeeper.NewWasmSnapshotter(app.CommitMultiStore(), &app.WasmClientKeeper),
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
    if err := wasmkeeper.InitializePinnedCodes(ctx); err != nil {
      cmtos.Exit(fmt.Sprintf("failed initialize pinned codes %s", err))
    }
  }
}
```

## Keeper instantiation

When it comes to instantiating `08-wasm`'s keeper there are two recommended ways of doing it. Choosing one or the other will depend on whether the chain already integrates [`x/wasm`](https://github.com/CosmWasm/wasmd/tree/main/x/wasm) or not. Both available constructor functions accept a querier parameter that should implement the [`Querier` interface of `wasmvm`](https://github.com/CosmWasm/wasmvm/blob/v1.5.0/types/queries.go#L37). If `nil` is provided, then a default querier implementation is used that returns error for any query type.

### If `x/wasm` is present

If the chain where the module is integrated uses `x/wasm` then we recommend that both `08-wasm` and `x/wasm` share the same Wasm VM instance. Having two separate Wasm VM instances is still possible, but care should be taken to make sure that both instances do not share the directory when the VM stores blobs and various caches, otherwise unexpected behaviour is likely to happen.

In order to share the Wasm VM instance please follow the guideline below. Please note that this requires `x/wasm`v0.41 or above.

- Instantiate the Wasm VM in `app.go` with the parameters of your choice.
- [Create an `Option` with this Wasm VM instance](https://github.com/CosmWasm/wasmd/blob/db93d7b6c7bb6f4a340d74b96a02cec885729b59/x/wasm/keeper/options.go#L21-L25).
- Add the option created in the previous step to a slice and [pass it to the `x/wasm NewKeeper` constructor function](https://github.com/CosmWasm/wasmd/blob/db93d7b6c7bb6f4a340d74b96a02cec885729b59/x/wasm/keeper/keeper_cgo.go#L36).
- Pass the pointer to the Wasm VM instance to `08-wasm` [NewKeeperWithVM constructor function](https://github.com/cosmos/ibc-go/blob/c95c22f45cb217d27aca2665af9ac60b0d2f3a0c/modules/light-clients/08-wasm/keeper/keeper.go#L33-L38).

The code to set this up would look something like this:

```go
// app.go
import (
  ...
  "github.com/cosmos/cosmos-sdk/runtime"
  
  wasmvm "github.com/CosmWasm/wasmvm"
  wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
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

// This sample code uses also an implementation of the 
// wasmvm.Querier interface (querier). If nil is passed
// instead, then a default querier will be used that
// returns an error for all query types.
app.WasmClientKeeper = wasmkeeper.NewKeeperWithVM(
  appCodec,
  runtime.NewKVStoreService(keys[wasmtypes.StoreKey]),
  app.IBCKeeper.ClientKeeper,
  authtypes.NewModuleAddress(govtypes.ModuleName).String(),
  wasmer, // pass the Wasm VM instance to `08-wasm` keeper constructor
  querier,
)
...
```

### If `x/wasm` is not present

If the chain does not use [`x/wasm`](https://github.com/CosmWasm/wasmd/tree/main/x/wasm), even though it is still possible to use the method above from the previous section
(e.g. instantiating a Wasm VM in app.go an pass it to 08-wasm's [`NewKeeperWithVM` constructor function](https://github.com/cosmos/ibc-go/blob/c95c22f45cb217d27aca2665af9ac60b0d2f3a0c/modules/light-clients/08-wasm/keeper/keeper.go#L33-L38), since there would be no need in this case to share the Wasm VM instance with another module, you can use the [`NewKeeperWithConfig`` constructor function](https://github.com/cosmos/ibc-go/blob/c95c22f45cb217d27aca2665af9ac60b0d2f3a0c/modules/light-clients/08-wasm/keeper/keeper.go#L52-L57) and provide the Wasm VM configuration parameters of your choice instead. A Wasm VM instance will be created in`NewKeeperWithConfig`. The parameters that can set are:

- `DataDir` is the [directory for Wasm blobs and various caches](https://github.com/CosmWasm/wasmvm/blob/1638725b25d799f078d053391945399cb35664b1/lib.go#L25). In `wasmd` this is set to the [`wasm` folder under the home directory](https://github.com/CosmWasm/wasmd/blob/36416def20effe47fb77f29f5ba35a003970fdba/app/app.go#L578).
- `SupportedCapabilities` is a comma separated [list of capabilities supported by the chain](https://github.com/CosmWasm/wasmvm/blob/1638725b25d799f078d053391945399cb35664b1/lib.go#L26). [`wasmd` sets this to all the available capabilities](https://github.com/CosmWasm/wasmd/blob/36416def20effe47fb77f29f5ba35a003970fdba/app/app.go#L586), but 08-wasm only requires `iterator`.
- `MemoryCacheSize` sets [the size in MiB of an in-memory cache for e.g. module caching](https://github.com/CosmWasm/wasmvm/blob/1638725b25d799f078d053391945399cb35664b1/lib.go#L29C16-L29C104). It is not consensus-critical and should be defined on a per-node basis, often in the range 100 to 1000 MB. [`wasmd` reads this value of](https://github.com/CosmWasm/wasmd/blob/36416def20effe47fb77f29f5ba35a003970fdba/app/app.go#L579). Default value is 256.
- `ContractDebugMode` is a [flag to enable/disable printing debug logs from the contract to STDOUT](https://github.com/CosmWasm/wasmvm/blob/1638725b25d799f078d053391945399cb35664b1/lib.go#L28). This should be false in production environments. Default value is false.

Another configuration parameter of the Wasm VM is the contract memory limit (in MiB), which is [set to 32](https://github.com/cosmos/ibc-go/blob/c95c22f45cb217d27aca2665af9ac60b0d2f3a0c/modules/light-clients/08-wasm/types/config.go#L5), [following the example of `wasmd`](https://github.com/CosmWasm/wasmd/blob/36416def20effe47fb77f29f5ba35a003970fdba/x/wasm/keeper/keeper.go#L32-L34). This parameter is not configurable by users of `08-wasm`.

The following sample code shows how the keeper would be constructed using this method:

```go
// app.go
import (
  ...
  "github.com/cosmos/cosmos-sdk/runtime"

  wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
  wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
  ...
)
...
wasmConfig := wasmtypes.WasmConfig{
  DataDir:               "ibc_08-wasm_client_data",
  SupportedCapabilities: "iterator",
  ContractDebugMode:     false,
}
app.WasmClientKeeper = wasmkeeper.NewKeeperWithConfig(
  appCodec,
  runtime.NewKVStoreService(keys[wasmtypes.StoreKey]),
  app.IBCKeeper.ClientKeeper, 
  authtypes.NewModuleAddress(govtypes.ModuleName).String(),
  wasmConfig,
  querier
)
```

Check out also the [`WasmConfig` type definition](https://github.com/cosmos/ibc-go/blob/c95c22f45cb217d27aca2665af9ac60b0d2f3a0c/modules/light-clients/08-wasm/types/config.go#L7-L20) for more information on each of the configurable parameters. Some parameters allow node-level configurations. There is additionally the function [`DefaultWasmConfig`](https://github.com/cosmos/ibc-go/blob/6d8cee53a72524b7cf396d65f6c19fed45803321/modules/light-clients/08-wasm/types/config.go#L30) available that returns a configuration with the default values.

## Updating `AllowedClients`

In order to use the `08-wasm` module chains must update the [`AllowedClients` parameter in the 02-client submodule](https://github.com/cosmos/ibc-go/blob/main/proto/ibc/core/client/v1/client.proto#L103) of core IBC. This can be configured directly in the application upgrade handler with the sample code below:

```go
params := clientKeeper.GetParams(ctx)
params.AllowedClients = append(params.AllowedClients, exported.Wasm)
clientKeeper.SetParams(ctx, params)
```

Or alternatively the parameter can be updated via a governance proposal (see at the bottom of section [`Creating clients`](../01-developer-guide/09-setup.md#creating-clients) for an example of how to do this).

## Adding snapshot support

In order to use the `08-wasm` module chains are required to register the `WasmSnapshotter` extension in the snapshot manager. This snapshotter takes care of persisting the external state, in the form of contract code, of the Wasm VM instance to disk when the chain is snapshotted. [This code](https://github.com/cosmos/ibc-go/blob/2bd29c08fd1fe50b461fc33a25735aa792dc896e/modules/light-clients/08-wasm/testing/simapp/app.go#L768-L776) should be placed in `NewSimApp` function in `app.go`:

## Pin byte codes at start

Wasm byte codes should be pinned to the WasmVM cache on every application start, therefore [this code](https://github.com/cosmos/ibc-go/blob/0ed221f687ffce75984bc57402fd678e07aa6cc5/modules/light-clients/08-wasm/testing/simapp/app.go#L821-L826) should be placed in `NewSimApp` function in `app.go`.
