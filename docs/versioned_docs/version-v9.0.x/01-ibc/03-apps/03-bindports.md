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

1. Add port ID as a key to the module store:

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

1. Add port ID to `x/<moduleName>/types/genesis.go`:

```go
// in x/<moduleName>/types/genesis.go

// DefaultGenesisState returns a GenesisState with "transfer" as the default PortID.
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

1. Bind to port(s) in the module keeper's `InitGenesis`:

```go
// InitGenesis initializes the ibc-module state and binds to PortID.
func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
  k.SetPort(ctx, state.PortId)

  // ...

  // Only try to bind to port if it is not already bound, since we may already own
  // port capability from capability InitGenesis
  if !k.hasCapability(ctx, state.PortId) {
    // transfer module binds to the transfer port on InitChain
    // and claims the returned capability
    err := k.BindPort(ctx, state.PortId)
    if err != nil {
      panic(fmt.Sprintf("could not claim port capability: %v", err))
    }
  }

  // ...
}
```

With:

```go
// IsBound checks if the  module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
  _, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
  return ok
}

// BindPort defines a wrapper function for the port Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
  cap := k.portKeeper.BindPort(ctx, portID)
  return k.ClaimCapability(ctx, cap, host.PortPath(portID))
}
```

The module binds to the desired port(s) and returns the capabilities.

In the above we find reference to keeper methods that wrap other keeper functionality, in the next section the keeper methods that need to be implemented will be defined.
