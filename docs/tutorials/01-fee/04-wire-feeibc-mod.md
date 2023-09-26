---
title: Wire Up the ICS-29 Fee Middleware to a Cosmos SDK Blockchain
sidebar_label: Wire Up the ICS-29 Fee Middleware to a Cosmos SDK Blockchain
sidebar_position: 4
slug: /fee/wire-feeibc-mod
---

import HighlightBox from '@site/src/components/HighlightBox';

# Wire Up the ICS-29 Fee Middleware to a Cosmos SDK Blockchain

<HighlightBox type="learning" title="Learning Goals">

In this section, you will:

- Add the ICS-29 Fee Middleware to a Cosmos SDK blockchain as a module.
- Wire up the ICS-29 Fee Middleware to the IBC transfer stack.

</HighlightBox>

## 1. Wire Up the ICS-29 Fee Middleware as a Cosmos SDK Module

The Fee Middleware is not just an IBC middleware, it is also a Cosmos SDK module since it manages its own state and defines its own messages.
We will first wire up the Fee Middleware as a Cosmos SDK module, then we will wire it up to the IBC transfer stack.

Cosmos SDK modules are registered in the `app/app.go` file. The `app.go` file is the entry point for the Cosmos SDK application. It is where the application is initialized and where the application's modules are registered.

We first need to import the `fee` module into the `app.go` file. bold and italic. Add the following import statements to the `app.go` file:

```go reference title="app/app.go"
https://github.com/srdtrk/ignite-fee-middleware-demo/blob/main/app/app.go#L99-L101
```

### 1.1. Add the Fee Middleware to the Module Managers and Define Its Account Permissions

Next, we need to add `fee` module to the module basic manager and define its account permissions. Add the following code to the `app.go` file:

```go title="app/app.go" {10,25}
	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		// ... other modules
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		ica.AppModuleBasic{},
		vesting.AppModuleBasic{},
+		ibcfee.AppModuleBasic{},
		consensus.AppModuleBasic{},
		// this line is used by starport scaffolding # stargate/app/moduleBasic
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		icatypes.ModuleName:            nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
+		ibcfeetypes.ModuleName:         nil,
		// this line is used by starport scaffolding # stargate/app/maccPerms
	}
```

Next, we need to add the fee middleware to the module manager. Add the following code to the `app.go` file:

```go title="app/app.go" {7}
	app.mm = module.NewManager(
		// ... other modules
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		params.NewAppModule(app.ParamsKeeper),
		transferModule,
		ibcfee.NewAppModule(app.IBCFeeKeeper),
		icaModule,
		// this line is used by starport scaffolding # stargate/app/appModule

		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
	)
```

### 1.2. Initialize the Fee Middleware Keeper

Next, we need to add the fee middleware keeper to the Cosmos App, register its store key, and initialize it.

```go title="app/app.go" {9}
// App extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type App struct {
	// ... other fields
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
+	IBCFeeKeeper          ibcfeekeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	// ... other fields
}
```

```go title="app/app.go" {7}
keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, authz.ModuleName, banktypes.StoreKey, stakingtypes.StoreKey,
		crisistypes.StoreKey, minttypes.StoreKey, distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, ibcexported.StoreKey, upgradetypes.StoreKey,
		feegrant.StoreKey, evidencetypes.StoreKey, ibctransfertypes.StoreKey, icahosttypes.StoreKey,
		capabilitytypes.StoreKey, group.StoreKey, icacontrollertypes.StoreKey, consensusparamtypes.StoreKey,
+		ibcfeetypes.StoreKey,
		// this line is used by starport scaffolding # stargate/app/storeKey
	)
```

Then initialize the keeper: 

:::warning

Make sure to do the following initialization after the `IBCKeeper` is initialized and before `TransferKeeper` is initialized, if you have done the changes above, then you may do the initialization at line 444.

:::

```go reference title="app/app.go"
https://github.com/srdtrk/ignite-fee-middleware-demo/blob/main/app/app.go#L452-L458
```

### 1.3. Add the Fee Middleware to SetOrderBeginBlockers, SetOrderEndBlockers, and genesisModuleOrder

Next, we need to add the fee middleware to the `SetOrderBeginBlockers`, `SetOrderEndBlockers`, and `genesisModuleOrder` functions. Add the following code to the `app.go` file:

```go title="app/app.go" {10,18,31}
	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	app.mm.SetOrderBeginBlockers(
		// upgrades should be run first
		upgradetypes.ModuleName,
		// ... other modules
		consensusparamtypes.ModuleName,
+		ibcfeetypes.ModuleName,
		// this line is used by starport scaffolding # stargate/app/beginBlockers
	)

	app.mm.SetOrderEndBlockers(
		// ... other modules
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
+		ibcfeetypes.ModuleName,
		// this line is used by starport scaffolding # stargate/app/endBlockers
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	genesisModuleOrder := []string{
		// ... other modules
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
+		ibcfeetypes.ModuleName,
		// this line is used by starport scaffolding # stargate/app/initGenesis
	}
	app.mm.SetOrderInitGenesis(genesisModuleOrder...)
	app.mm.SetOrderExportGenesis(genesisModuleOrder...)
```

## 2. Wire Up the ICS-29 Fee Middleware to the IBC Transfer Stack

### 2.1. Wire Up the ICS-29 Fee Middleware to the `TransferKeeper`

The ICS-29 Fee Middleware Keeper implements [`ICS4Wrapper`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/core/05-port/types/module.go#L109-L133) interface. This means that the `IBCFeeKeeper` wraps the `IBCKeeper.ChannelKeeper` and that it can replace the use of the `ChannelKeeper` for sending packets, writing acknowledgements, and retrieving the IBC channel version.

We need to replace the `ChannelKeeper` with the `IBCFeeKeeper` in the `TransferKeeper`. To do this, we need to modify the `TransferKeeper` initialization in the `app.go` file.

```go title="app/app.go" {6,7}
	// Create Transfer Keepers
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		app.GetSubspace(ibctransfertypes.ModuleName),
-		app.IBCKeeper.ChannelKeeper,
+		app.IBCFeeKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		scopedTransferKeeper,
	)
```

### 2.2. Wire Up the ICS-29 Fee Middleware to the `TransferModule`

Currently, the IBC Transfer stack does not exist in `app/app.go`. What we have are the transfer module (which is a Cosmos SDK module) and the transfer IBC module (which is an IBC application).

```go reference title="app/app.go"
https://github.com/srdtrk/cosmoverse2023-ibc-fee-demo/blob/0f41b3c6b4e065aa1a860de3e3038d489c37a28a/app/app.go#L457-L458
```

Instead we need to "convert" the transfer IBC module to an IBC application stack that includes both the transfer IBC module and the ICS-29 Fee Middleware. Modify the `app.go` file as follows:

```go title="app/app.go" {2-6}
	transferModule := transfer.NewAppModule(app.TransferKeeper)
-	transferIBCModule := transfer.NewIBCModule(app.TransferKeeper)
+	/**** IBC Transfer Stack ****/
+	var transferStack ibcporttypes.IBCModule
+	transferStack = transfer.NewIBCModule(app.TransferKeeper)
+	transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)
```

And finally, we need to add the `transferStack` to the `ibcRouter`. Modify the `app.go` file as follows:

```go title="app/app.go" {2-3}
	ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostIBCModule).
-		AddRoute(ibctransfertypes.ModuleName, transferIBCModule)
+		AddRoute(ibctransfertypes.ModuleName, transferStack)
```

This completes the wiring of the ICS-29 Fee Middleware to the IBC transfer stack! See a full example of the `app.go` file [here](//TODO: add link). Test that the application is still running with `ignite chain serve --reset-once`, and quit with `q`.
