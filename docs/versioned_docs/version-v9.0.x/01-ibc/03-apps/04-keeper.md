---
title: Keeper
sidebar_label: Keeper
sidebar_position: 4
slug: /ibc/apps/keeper
---

# Keeper

:::note Synopsis
Learn how to implement the IBC Module keeper.
:::

:::note

## Pre-requisite readings

- [IBC Overview](../01-overview.md)
- [IBC default integration](../02-integration.md)

:::
In the previous sections, on channel handshake callbacks and port binding in `InitGenesis`, a reference was made to keeper methods that need to be implemented when creating a custom IBC module. Below is an overview of how to define an IBC module's keeper.

> Note that some code has been left out for clarity, to get a full code overview, please refer to [the transfer module's keeper in the ibc-go repo](https://github.com/cosmos/ibc-go/blob/main/modules/apps/transfer/keeper/keeper.go).

```go
// Keeper defines the IBC app module keeper
type Keeper struct {
  storeKey   sdk.StoreKey
  cdc        codec.BinaryCodec
  paramSpace paramtypes.Subspace

  channelKeeper types.ChannelKeeper
  portKeeper    types.PortKeeper

  // ... additional according to custom logic
}

// NewKeeper creates a new IBC app module Keeper instance
func NewKeeper(
  // args
) Keeper {
  // ...

  return Keeper{
    cdc:           cdc,
    storeKey:      key,
    paramSpace:    paramSpace,

    channelKeeper: channelKeeper,
    portKeeper:    portKeeper,
    scopedKeeper:  scopedKeeper,

    // ... additional according to custom logic
  }
}

// GetPort returns the portID for the IBC app module. Used in ExportGenesis
func (k Keeper) GetPort(ctx sdk.Context) string {
  store := ctx.KVStore(k.storeKey)
  return string(store.Get(types.PortKey))
}

// SetPort sets the portID for the IBC app module. Used in InitGenesis
func (k Keeper) SetPort(ctx sdk.Context, portID string) {
  store := ctx.KVStore(k.storeKey)
  store.Set(types.PortKey, []byte(portID))
}

// ... additional according to custom logic
```
