# Migrating from v7 to v7.1

This guide provides instructions for migrating to a new version of ibc-go.

There are four sections based on the four potential user groups of this document:

- [Chains](#chains)
- [IBC Apps](#ibc-apps)
- [Relayers](#relayers)
- [IBC Light Clients](#ibc-light-clients)

**Note:** ibc-go supports golang semantic versioning and therefore all imports must be updated on major version releases.

## Chains

- No relevant changes were made in this release.

## IBC Apps

- No relevant changes were made in this release.

## Relayers

The event attribute `packet_connection` (`connectiontypes.AttributeKeyConnection`) has been deprecated. 
Please use the `connection_id` attribute (`connectiontypes.AttributeKeyConnectionID`) which is emitted by all channel events.
Only send packet, receive packet, write acknowledgement, and acknowledge packet events used `packet_connection` previously.

## IBC Light Clients

- No relevant changes were made in this release.
