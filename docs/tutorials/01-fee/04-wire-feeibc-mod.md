---
title: Wire Up the ICS-29 Fee Middleware to a Cosmos SDK Blockchain
sidebar_label: Wire Up the ICS-29 Fee Middleware to a Cosmos SDK Blockchain
sidebar_position: 4
slug: /fee/wire-feeibc-mod
---

import HighlightBox from '@site/src/components/HighlightBox';

# Wire up the ICS-29 Fee Middleware to a Cosmos SDK blockchain

<HighlightBox type="learning" title="Learning Goals">

In this section, you will:

- Add the ICS-29 Fee Middleware to a Cosmos SDK blockchain as a module.
- Wire up the ICS-29 Fee Middleware to the IBC transfer stack.

</HighlightBox>

## 1. Wire up the ICS-29 Fee Middleware as a Cosmos SDK module

The Fee Middleware is not just an IBC middleware, it is also a Cosmos SDK module since it manages its own state and defines its own messages.
We will first wire up the Fee Middleware as a Cosmos SDK module, then we will wire it up to the IBC transfer stack.

Cosmos SDK modules are registered in the `app/app.go` file. The `app.go` file is the entry point for the Cosmos SDK application. It is where the application is initialized and where the application's modules are registered.

We first need to import the `fee` module into the `app.go` file. Add the following import statements to the `app.go` file:

```go reference title="app/app.go"
https://github.com/srdtrk/cosmoverse2023-ibc-fee-demo/blob/64e572214b4ba9a1075db96440dd83d4b90a6052/app/app.go#L99-L101
```

### 1.1. Add the Fee Middleware to the module managers and define its account permissions

Next, we need to add `fee` module to the module basic manager and define its account permissions. Add the following code to the `app.go` file:

```go title="app/app.go"
	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		// ... other modules
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		ica.AppModuleBasic{},
		vesting.AppModuleBasic{},
		// plus-diff-line 
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
		// plus-diff-line 
+		ibcfeetypes.ModuleName:         nil,
		// this line is used by starport scaffolding # stargate/app/maccPerms
	}
```

Next, we need to add the fee middleware to the module manager. Add the following code to the `app.go` file:

```go title="app/app.go"
	app.mm = module.NewManager(
		// ... other modules
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		params.NewAppModule(app.ParamsKeeper),
		transferModule,
		// plus-diff-line 
+		ibcfee.NewAppModule(app.IBCFeeKeeper),
		icaModule,
		// this line is used by starport scaffolding # stargate/app/appModule

		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
	)
```

Note that we have added `ibcfee.NewAppModule(app.IBCFeeKeeper)` to the module manager but we have not yet created nor initialized the `app.IBCFeeKeeper`. We will do that next.

### 1.2. Initialize the Fee Middleware keeper

Next, we need to add the fee middleware keeper to the Cosmos App, register its store key, and initialize it.

```go title="app/app.go"
// App extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type App struct {
	// ... other fields
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	// plus-diff-line 
+	IBCFeeKeeper          ibcfeekeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	// ... other fields
}
```

```go title="app/app.go"
keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, authz.ModuleName, banktypes.StoreKey, stakingtypes.StoreKey,
		crisistypes.StoreKey, minttypes.StoreKey, distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, ibcexported.StoreKey, upgradetypes.StoreKey,
		feegrant.StoreKey, evidencetypes.StoreKey, ibctransfertypes.StoreKey, icahosttypes.StoreKey,
		capabilitytypes.StoreKey, group.StoreKey, icacontrollertypes.StoreKey, consensusparamtypes.StoreKey,
		// plus-diff-line 
+		ibcfeetypes.StoreKey,
		// this line is used by starport scaffolding # stargate/app/storeKey
	)
```

Then initialize the keeper: 

:::warning

Make sure to do the following initialization after the `IBCKeeper` is initialized and before `TransferKeeper` is initialized.

:::

```go reference title="app/app.go"
https://github.com/srdtrk/cosmoverse2023-ibc-fee-demo/blob/64e572214b4ba9a1075db96440dd83d4b90a6052/app/app.go#L452-L458
```

### 1.3. Add the Fee Middleware to SetOrderBeginBlockers, SetOrderEndBlockers, and genesisModuleOrder

Next, we need to add the fee middleware to the `SetOrderBeginBlockers`, `SetOrderEndBlockers`, and `genesisModuleOrder` functions. Add the following code to the `app.go` file:

```go title="app/app.go"
	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	app.mm.SetOrderBeginBlockers(
		// ... other modules
		icatypes.ModuleName,
		// plus-diff-line
+		ibcfeetypes.ModuleName,
		genutiltypes.ModuleName,
		// ... other modules
		consensusparamtypes.ModuleName,
		// this line is used by starport scaffolding # stargate/app/beginBlockers
	)

	app.mm.SetOrderEndBlockers(
		// ... other modules
		icatypes.ModuleName,
		// plus-diff-line
+		ibcfeetypes.ModuleName,
		capabilitytypes.ModuleName,
		// ... other modules
		consensusparamtypes.ModuleName,
		// this line is used by starport scaffolding # stargate/app/endBlockers
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	genesisModuleOrder := []string{
		// ... other modules
		icatypes.ModuleName,
		// plus-diff-line
+		ibcfeetypes.ModuleName,
		evidencetypes.ModuleName,
		// ... other modules
		consensusparamtypes.ModuleName,
		// this line is used by starport scaffolding # stargate/app/initGenesis
	}
	app.mm.SetOrderInitGenesis(genesisModuleOrder...)
	app.mm.SetOrderExportGenesis(genesisModuleOrder...)
```

## 2. Wire up the ICS-29 Fee Middleware to the IBC Transfer stack

### 2.1. Wire up the ICS-29 Fee Middleware to the `TransferKeeper`

The ICS-29 Fee Middleware Keeper implements [`ICS4Wrapper`](https://github.com/cosmos/ibc-go/blob/v7.3.0/modules/core/05-port/types/module.go#L109-L133) interface. This means that the `IBCFeeKeeper` wraps the `IBCKeeper.ChannelKeeper` and that it can replace the use of the `ChannelKeeper` for sending packets, writing acknowledgements, and retrieving the IBC channel version.

We need to replace the `ChannelKeeper` with the `IBCFeeKeeper` in the `TransferKeeper`. To do this, we need to modify the `TransferKeeper` initialization in the `app.go` file.

```go title="app/app.go"
	// Create Transfer Keepers
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		app.GetSubspace(ibctransfertypes.ModuleName),
		// minus-diff-line 
-		app.IBCKeeper.ChannelKeeper,
		// plus-diff-line 
+		app.IBCFeeKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		scopedTransferKeeper,
	)
```

### 2.2. Wire up the ICS-29 Fee Middleware to the `TransferModule`

Currently, our `app/app.go` only contains the transfer module, which is a regular SDK AppModule (that manages state and has its own messages) that also fulfills the `IBCModule` interface and therefore has the ability to handle both channel handshake and packet lifecycle callbacks.

:::note

The transfer module is instantiated two times, once as a regular SDK module and once as an IBC module.

```go reference title="app/app.go"
https://github.com/srdtrk/cosmoverse2023-ibc-fee-demo/blob/0f41b3c6b4e065aa1a860de3e3038d489c37a28a/app/app.go#L457-L458
```

:::

We therefore need to "convert" the `transferIBCModule` to an IBC application stack that includes both the `transferIBCModule` and the ICS-29 Fee Middleware. Modify the `app.go` file as follows:

```go title="app/app.go"
	transferModule := transfer.NewAppModule(app.TransferKeeper)
	// minus-diff-line 
-	transferIBCModule := transfer.NewIBCModule(app.TransferKeeper)
	// plus-diff-start
+
+	/**** IBC Transfer Stack ****/
+	var transferStack ibcporttypes.IBCModule
+	transferStack = transfer.NewIBCModule(app.TransferKeeper)
+	transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)
	// plus-diff-end 
```

And finally, we need to add the `transferStack` to the `ibcRouter`. Modify the `app.go` file as follows:

```go title="app/app.go"
	ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostIBCModule).
		// minus-diff-line 
-		AddRoute(ibctransfertypes.ModuleName, transferIBCModule)
		// plus-diff-line 
+		AddRoute(ibctransfertypes.ModuleName, transferStack)
```

This completes the wiring of the ICS-29 Fee Middleware to the IBC transfer stack! See a full example of the `app.go` file with the fee middleware wired up [here](https://github.com/srdtrk/cosmoverse2023-ibc-fee-demo/blob/64e572214b4ba9a1075db96440dd83d4b90a6052/app/app.go) and the diff [here](https://github.com/srdtrk/cosmoverse2023-ibc-fee-demo/commit/64e572214b4ba9a1075db96440dd83d4b90a6052). Test that the application is still running with `ignite chain serve --reset-once`, and quit with `q`.
