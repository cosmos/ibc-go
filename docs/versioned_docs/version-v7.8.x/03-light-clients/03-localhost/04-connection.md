---
title: Connection
sidebar_label: Connection
sidebar_position: 4
slug: /ibc/light-clients/localhost/connection
---


# Localhost connections

The 09-localhost light client module integrates with core IBC through a single sentinel localhost connection.
The sentinel `ConnectionEnd` is stored by default in the core IBC store.

This enables channel handshakes to be initiated out of the box by supplying the localhost connection identifier (`connection-localhost`) in the `connectionHops` parameter of `MsgChannelOpenInit`.

The `ConnectionEnd` is created and set in store via the `InitGenesis` handler of the 03-connection submodule in core IBC.
The `ConnectionEnd` and its `Counterparty` both reference the `09-localhost` client identifier, and share the localhost connection identifier `connection-localhost`.

```go
// CreateSentinelLocalhostConnection creates and sets the sentinel localhost connection end in the IBC store.
func (k Keeper) CreateSentinelLocalhostConnection(ctx sdk.Context) {
  counterparty := types.NewCounterparty(exported.LocalhostClientID, exported.LocalhostConnectionID, commitmenttypes.NewMerklePrefix(k.GetCommitmentPrefix().Bytes()))
  connectionEnd := types.NewConnectionEnd(types.OPEN, exported.LocalhostClientID, counterparty, types.ExportedVersionsToProto(types.GetCompatibleVersions()), 0)

  k.SetConnection(ctx, exported.LocalhostConnectionID, connectionEnd)
}
```

Note that connection handshakes are disallowed when using the `09-localhost` client type.
