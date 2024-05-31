---
title: Client State interface
sidebar_label: Client State interface
sidebar_position: 3
slug: /ibc/light-clients/client-state
---


# Implementing the `ClientState` interface

Learn how to implement the [`ClientState`](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/core/exported/client.go#L36) interface.

## `ClientType` method

`ClientType` should return a unique string identifier of the light client. This will be used when generating a client identifier.
The format is created as follows: `{client-type}-{N}` where `{N}` is the unique global nonce associated with a specific client (e.g `07-tendermint-0`).

## `Validate` method

`Validate` should validate every client state field and should return an error if any value is invalid. The light client
implementer is in charge of determining which checks are required. See the [Tendermint light client implementation](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/light-clients/07-tendermint/client_state.go#L111) as a reference.
