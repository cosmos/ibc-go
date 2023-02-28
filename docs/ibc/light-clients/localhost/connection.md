<!--
order: 4
-->

# Localhost connections

The 09-localhost light client module integrates with core IBC through a single sentinel localhost connection.
The sentinel `ConnectionEnd` is stored by default in the core IBC store.

This enables channel handshakes to be initiated out of the box by supplying the localhost connection identifier (`connection-localhost`) in the `connectionHops` parameter of `MsgChannelOpenInit`.

The `ConnectionEnd` is created and set in store via the `InitGenesis` handler of the 03-connection submodule in core IBC.
The `ConnectionEnd` and its `Counterparty` both reference the `09-localhost` client identifier, and share the localhost connection identifier `connection-localhost`.

```go
func (k Keeper) CreateSentinelLocalhostConnection() types.ConnectionEnd {
  counterparty := types.NewCounterparty(exported.Localhost, types.LocalhostID, commitmenttypes.NewMerklePrefix(k.GetCommitmentPrefix().Bytes()))
  return types.NewConnectionEnd(types.OPEN, exported.Localhost, counterparty, types.ExportedVersionsToProto(types.GetCompatibleVersions()), 0)
}
```
