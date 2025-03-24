---
title: Bind ports
sidebar_label: Bind ports
sidebar_position: 3
slug: /ibc/apps/bindports
---

# Bind ports

:::note Synopsis
Learn what changes to make to bind modules to their ports on initialization.
:::

:::note

## Pre-requisite readings

- [IBC Overview](../01-overview.md)
- [IBC default integration](../02-integration.md)

:::
Currently, ports must be bound on app initialization. In order to bind modules to their respective ports on initialization, the following needs to be implemented:

> Note that `portID` does not refer to a certain numerical ID, like `localhost:8080` with a `portID` 8080. Rather it refers to the application module the port binds. For IBC Modules built with the Cosmos SDK, it defaults to the module's name and for Cosmwasm contracts it defaults to the contract address.

1. Add port ID to the `GenesisState` proto definition:

```protobuf
message GenesisState {
  string port_id = 1;
  // other fields
}
```

2. Add port ID as a key to the module store:

```go
// x/<moduleName>/types/keys.go
const (
  // ModuleName defines the IBC Module name
  ModuleName = "moduleName"

  // Version defines the current version the IBC
  // module supports
  Version = "moduleVersion-1"

  // PortID is the default port id that module binds to
  PortID = "portID"

  // ...
)
```

3. Add port ID to `x/<moduleName>/types/genesis.go`:

```go
// in x/<moduleName>/types/genesis.go

// DefaultGenesisState returns a GenesisState with "portID" as the default PortID.
func DefaultGenesisState() *GenesisState {
  return &GenesisState{
    PortId:      PortID,
    // additional k-v fields
  }
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
  if err := host.PortIdentifierValidator(gs.PortId); err != nil {
    return err
  }
  //additional validations

  return gs.Params.Validate()
}
```

4. Set the port in the module keeper's for `InitGenesis`:

:::note
The capability module has been removed so port binding has also changed
:::

```go
// SetPort sets the portID for the transfer module. Used in InitGenesis
func (k Keeper) SetPort(ctx sdk.Context, portID string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.PortKey, []byte(portID)); err != nil {
		panic(err)
	}
}

  // Initialize any other module state, like params with SetParams.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&params)
	if err := store.Set([]byte(types.ParamsKey), bz); err != nil {
		panic(err)
	}
}
  // ...

```

The module is set to the desired port. The setting and sealing happens during creation of the IBC router. 
