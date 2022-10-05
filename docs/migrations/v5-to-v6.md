
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

## IBC Apps

### ICS04 - SendPacket API change

The `SendPacket` API has been simplified:

```diff
// SendPacket is called by a module in order to send an IBC packet on a channel
 func (k Keeper) SendPacket(
        ctx sdk.Context,
        channelCap *capabilitytypes.Capability,
-       packet exported.PacketI,
-) error {
+       sourcePort string,
+       sourceChannel string,
+       timeoutHeight clienttypes.Height,
+       timeoutTimestamp uint64,
+       data []byte,
+) (uint64, error) {
```

Callers no longer need to pass in a pre-constructed packet. 
The destination port/channel identifiers and the packet sequence will be determined by core IBC.
`SendPacket` will return the packet sequence.
