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

- You should pass the `authority` to the icahost keeper. ([#3520](https://github.com/cosmos/ibc-go/pull/3520)) See [diff](https://github.com/cosmos/ibc-go/pull/3520/files#diff-d18972debee5e64f16e40807b2ae112ddbe609504a93ea5e1c80a5d489c3a08a).

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

## IBC Apps

TODO: https://github.com/cosmos/ibc-go/pull/3303

## Relayers

- No relevant changes were made in this release.

## IBC Light Clients

- No relevant changes were made in this release.
