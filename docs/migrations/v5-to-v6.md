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

The `ibc-go/v6` release introduces a new set of migrations for ICS27 interchain accounts. Ownership of ICS27 channel capabilities is tranferred from authentication modules and will now reside with the ICS27 `controller` submodule moving forward. 

For chains which implement custom authentication modules using the ICS27 `controller` submodule this requires a migration function to be included in the application upgrade handler. A subsequent migration handler is run automatically, asserting the ownership of ICS27 channel capabilities has been transferred successfully.

This migration facilitates the addition of the ICS27 `controller` submodule `Msg` server which provides a standardised approach to integrating existing forms of authentication such as `x/gov` and `x/group` provided by the Cosmos SDK. 

[comment]: <> (TODO: update ADR009 PR link when merged)
For more information please refer to [ADR 009](https://github.com/cosmos/ibc-go/pull/2218).

#### Upgrade Proposal

Please refer to [PR #2383](https://github.com/cosmos/ibc-go/pull/2383) for integrating the ICS27 channel capability migration logic or follow the steps outlined below:

1. Add the upgrade migration logic to chain distribution. This may be, for example, maintained under `app/upgrades/v6`:

```go
package v6

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	v6 "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/migrations/v6"
)

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
        "ics27-auth-module",
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
