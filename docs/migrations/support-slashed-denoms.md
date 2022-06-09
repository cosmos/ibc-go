# Migrating from Unsupported Base Denom Slashes to Supporting Slashed BaseDenoms

This document is intended to highlight significant changes which may require more information than presented in the CHANGELOG.
Any changes that must be done by a user of ibc-go should be documented here.

There are four sections based on the four potential user groups of this document:
- Chains
- IBC Apps
- Relayers
- IBC Light Clients

**Note:** ibc-go supports golang semantic versioning and therefore all imports must be updated to bump the version number on major releases.
```go
github.com/cosmos/ibc-go/v3 -> github.com/cosmos/ibc-go/v4
```

This document is necessary when chains are upgrading from a version that does not support slashed base denoms (e.g. v3.0.0) to a version that does (e.g. v3.1.0).

If a chain receives a slashed denom before it upgrades to supporting it, the receive may pass however the trace information will be incorrect.

E.g. If a base denom of `testcoin/testcoin/testcoin` is sent to a chain that does not support slashes in the base denom; the receive will be successful. However, the trace information stored on the receiving chain will be: `Trace: "testcoin/testcoin", BaseDenom: "testcoin"`

This incorrect trace information must be corrected when the chain does upgrade to fully supporting slashed denominations.

To do so, chain binaries should include a migration script that will run when the chain upgrades from not supporting slashed base denominations to supporting slashed base denominations.

## Chains

### Transfer

The transfer module will now support slashes in base denoms, so we must iterate over current traces to check if any of them are incorrectly formed and correct the trace information.

### Upgrade Propsoal

```go
app.UpgradeKeeper.SetUpgradeHandler("v3.1.0",
    func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
        // list of traces that must replace the old traces in store
        var newTraces []transfertypes.DenomTrace

        transferKeeper.IterateDenomTraces(ctx,
        func(dt transfertypes.DenomTrace) bool {
            // check if the new way of splitting FullDenom
            // into Trace and BaseDenom is the same as the current
            // DenomTrace.
            // If it isn't then store the new DenomTrace in the list of new traces.
            newTrace := transfertypes.ParseDenomTrace(dt.GetFullDenomPath())

            if !reflect.DeepEqual(newTrace, dt) {
                append(newTraces, newTrace)
            }
        })

        // replace the outdated traces with the new trace information
        for _, nt := range newTraces {
            transferKeeper.SetDenomTrace(ctx, nt)
        }
    }
)
```

This is only necessary if there are DenomTraces in the store with incorrect trace information from previously received coins that had a slash in the base denom. However, it is recommended that any chain upgrading to support slashed denominations runs this code for safety.

#### Add `StoreUpgrades` for Transfer module

For Transfer it is also necessary to [manually add store upgrades](https://docs.cosmos.network/v0.44/core/upgrade.html#add-storeupgrades-for-new-modules) for the transfer module and then configure the store loader to apply those upgrades in `app.go` if you wish to use the upgrade handler method above.

```go
// Here the upgrade name is just an example
if upgradeInfo.Name == "supportSlashingDenomUpgrade" && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height)  {
    storeUpgrades := store.StoreUpgrades{
        Added: []string{transfertypes.StoreKey}
    }

    app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
}
```

This ensures that the transfer module's stores are added to the multistore before the migrations begin. 

### Genesis Migration

If the chain chooses to add support for slashes in base denoms via genesis export, then the trace information must be corrected during genesis migration.

The migration code required may look like:

```go
func MigrateGenesis(appState genutiltypes.AppMap, clientCtx client.Context, genDoc tmtypes.GenesisDoc) (genutiltypes.AppMap, error) {
    if appState[transfertypes.ModuleName] != nil {
        transferGenState := &transfertypes.GenesisState
        clientCtx.JSONCodec.MustUnmarshalJSON(appState[transfertypes.ModuleName], transferGenState)

        substituteTraces := make([]transfertypes.DenomTrace, len(transferGenState.Traces)
        for i, dt := range transferGenState.Traces {
            // replace all previous traces with the latest trace
            // note most traces will have same value
            newTrace := transfertypes.ParseDenomTrace(dt.GetFullDenomPath())

            subsituteTraces[i] = newTrace
        }

        transferGenState.Traces = substituteTraces

        // delete old genesis state
		delete(appState, transfertypes.ModuleName)

        // set new ibc transfer genesis state
		appState[transfertypes.ModuleName] = clientCtx.JSONCodec.MustMarshalJSON(transferGenState)
    }

    return appState, nil
}
```