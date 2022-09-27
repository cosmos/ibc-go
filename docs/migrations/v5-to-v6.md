# Migrating from ibc-go v5 to v6

This document is intended to highlight significant changes which may require more information than presented in the CHANGELOG.
Any changes that must be done by a user of ibc-go should be documented here.

There are four sections based on the four potential user groups of this document:
- Chains
- IBC Apps
- Relayers
- IBC Light Clients

**Note:** ibc-go supports golang semantic versioning and therefore all imports must be updated to bump the version number on major releases.

## Chains

- No relevant changes were made in this release.

## IBC Apps

### ICS27 - Interchain Accounts

#### Upgrade Proposal

The `ibc-go/v6` release introduces a migration for ICS27 interchain accounts whereby ownership of channel capabilities is transferred from base applications previously referred to as authentication modules to the ICS27 controller submodule. This coincides with the introduction of the ICS27 `controller` submodule `Msg` service which provides a standardised approach to integrating existing forms of authentication such as `x/gov` and `x/group` provided by the Cosmos SDK.

Please refer to the following PR diff for integrating the ICS27 channel capability migration logic:

- https://github.com/cosmos/ibc-go/pull/2383

Add the upgrade logic to chain distribution:

```go
const (
	UpgradeName = "v6"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	cdc codec.BinaryCodec,
	capabilityStoreKey *storetypes.KVStoreKey,
	capabilityKeeper *capabilitykeeper.Keeper,
	moduleName string,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		if err := v6.MigrateICS27ChannelCapability(ctx, cdc, capabilityStoreKey, capabilityKeeper, moduleName); err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
```

Set the upgrade handler in `app.go`:

```go
app.UpgradeKeeper.SetUpgradeHandler(
	v6.UpgradeName,
	v6.CreateUpgradeHandler(
        app.mm, 
        app.configurator, 
        app.appCodec, 
        app.keys[capabilitytypes.ModuleName], 
        app.CapabilityKeeper, 
        ibcmock.ModuleName+icacontrollertypes.SubModuleName,
    ),
)
```

---

### TODO Genesis types docs
The ICS27 genesis types have been moved to their own package:

```
option go_package = "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/genesis/types";
```

---

The ICS27 `host` submodule `NewKeeper` function in `modules/apps/27-interchain-acccounts/host/keeper` now includes an additional parameter of type `ICS4Wrapper`.
This provides the `host` submodule with the ability to correctly unwrap channel versions in the event of a channel reopening handshake.

```diff
func NewKeeper(
	cdc codec.BinaryCodec, key storetypes.StoreKey, paramSpace paramtypes.Subspace,
-	channelKeeper icatypes.ChannelKeeper, portKeeper icatypes.PortKeeper,
+	ics4Wrapper icatypes.ICS4Wrapper, channelKeeper icatypes.ChannelKeeper, portKeeper icatypes.PortKeeper,
	accountKeeper icatypes.AccountKeeper, scopedKeeper icatypes.ScopedKeeper, msgRouter icatypes.MessageRouter,
) Keeper
```

The `msgRouter` parameter has also been updated to accept a type which fulfills the `MessageRouter` interface as defined in `27-interchain-accounts/types`.

```go
type MessageRouter interface {
	Handler(msg sdk.Msg) baseapp.MsgServiceHandler
}
```

## Relayers

- No relevant changes were made in this release.

## IBC Light Clients

- No relevant changes were made in this release.
