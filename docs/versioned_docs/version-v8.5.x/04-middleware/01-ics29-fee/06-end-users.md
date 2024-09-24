---
title: End Users
sidebar_label: End Users
sidebar_position: 6
slug: /middleware/ics29-fee/end-users
---

# For end users

:::note Synopsis
Learn how to incentivize IBC packets using the ICS29 Fee Middleware module.
:::

:::note

## Pre-requisite readings

- [Fee Middleware](01-overview.md)

:::

## Summary

Different types of end users:

- CLI users who want to manually incentivize IBC packets
- Client developers

The Fee Middleware module allows end users to add a 'tip' to each IBC packet which will incentivize relayer operators to relay packets between chains. gRPC endpoints are exposed for client developers as well as a simple CLI for manually incentivizing IBC packets.

## CLI Users

For an in depth guide on how to use the ICS29 Fee Middleware module using the CLI please take a look at the [wiki](https://github.com/cosmos/ibc-go/wiki/Fee-enabled-fungible-token-transfers#asynchronous-incentivization-of-a-fungible-token-transfer) on the `ibc-go` repo.

## Client developers

Client developers can read more about the relevant ICS29 message types in the [Fee messages section](03-msgs.md).

[CosmJS](https://github.com/cosmos/cosmjs) is a useful client library for signing and broadcasting Cosmos SDK messages.
