# Migrating from v7 to v8

This guide provides instructions for migrating to version `v8.0.0` of ibc-go.

There are four sections based on the four potential user groups of this document:

- [Migrating from v7 to v8](#migrating-from-v7-to-v8)
	- [Chains](#chains)
	- [IBC Apps](#ibc-apps)
	- [Relayers](#relayers)
	- [IBC Light Clients](#ibc-light-clients)

**Note:** ibc-go supports golang semantic versioning and therefore all imports must be updated on major version releases.

## Chains

TODO: https://github.com/cosmos/ibc-go/pull/3505 (extra parameter added to transfer's `GenesisState`)

- You must pass the `authority` to the icahost keeper. ([#3520](https://github.com/cosmos/ibc-go/pull/3520)) See [diff](https://github.com/cosmos/ibc-go/pull/3520/files#diff-d18972debee5e64f16e40807b2ae112ddbe609504a93ea5e1c80a5d489c3a08a).

```diff
// app.go

	// ICA Host keeper
	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec, keys[icahosttypes.StoreKey], app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCFeeKeeper, // use ics29 fee as ics4Wrapper in middleware stack
		app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
		app.AccountKeeper, scopedICAHostKeeper, app.MsgServiceRouter(),
+		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
```

- You must pass the `authority` to the icacontroller keeper. ([#3590](https://github.com/cosmos/ibc-go/pull/3590)) See [diff](https://github.com/cosmos/ibc-go/pull/3590/files#diff-d18972debee5e64f16e40807b2ae112ddbe609504a93ea5e1c80a5d489c3a08a).

```diff
// app.go

	// ICA Controller keeper
	app.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
		appCodec, keys[icacontrollertypes.StoreKey], app.GetSubspace(icacontrollertypes.SubModuleName),
		app.IBCFeeKeeper, // use ics29 fee as ics4Wrapper in middleware stack
		app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
		scopedICAControllerKeeper, app.MsgServiceRouter(),
+		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
```

- You must pass the `authority` to the ibctransfer keeper. ([#3553](https://github.com/cosmos/ibc-go/pull/3553)) See [diff](https://github.com/cosmos/ibc-go/pull/3553/files#diff-d18972debee5e64f16e40807b2ae112ddbe609504a93ea5e1c80a5d489c3a08a).

```diff
// app.go

	// Create Transfer Keeper and pass IBCFeeKeeper as expected Channel and PortKeeper
	// since fee middleware will wrap the IBCKeeper for underlying application.
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec, keys[ibctransfertypes.StoreKey], app.GetSubspace(ibctransfertypes.ModuleName),
		app.IBCFeeKeeper, // ISC4 Wrapper: fee IBC middleware
		app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
		app.AccountKeeper, app.BankKeeper, scopedTransferKeeper,
+		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
```

- You should pass the `authority` to the IBC keeper. ([#3640](https://github.com/cosmos/ibc-go/pull/3640) and [#3650](https://github.com/cosmos/ibc-go/pull/3650)) See [diff](https://github.com/cosmos/ibc-go/pull/3640/files#diff-d18972debee5e64f16e40807b2ae112ddbe609504a93ea5e1c80a5d489c3a08a).

```diff
// app.go

	// IBC Keepers
	app.IBCKeeper = ibckeeper.NewKeeper(
-       appCodec, keys[ibcexported.StoreKey], app.GetSubspace(ibcexported.ModuleName), app.StakingKeeper, app.UpgradeKeeper, scopedIBCKeeper,
+		appCodec, keys[ibcexported.StoreKey], app.GetSubspace(ibcexported.ModuleName), app.StakingKeeper, app.UpgradeKeeper, scopedIBCKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
```

## IBC Apps

TODO: https://github.com/cosmos/ibc-go/pull/3303

## Relayers

- No relevant changes were made in this release.

## IBC Light Clients

- No relevant changes were made in this release.
