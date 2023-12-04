<!-- More detailed information about the requirements engineering process can be found at https://github.com/cosmos/ibc-go/wiki/Requirements-engineering -->

# Business requirements

The localhost client provides a unified interface to interact with different applications on a single chain.

## Problem

Currently applications, or smart contracts, on a single chain cannot communicate through a single interface. This means a user must interact with each application or smart contract directly which can be cumbersome, especially when a user wants to interact with many applications on one chain.

## Objectives

To provide a single interface in line with IBC application layer semantics, that enables a user to interact with multiple different applications or smart contracts remotely on a given blockchain. 

## Scope

| Features  | Release |
| --------- | ------- |
| To interface with applications or smart contracts on a blockchain through the IBC application layer | v7.1.0 |
| To use the localhost client without requiring third party relayer infrastructure | N/A |

# User requirements

## Use cases

### Interacting with Smart Contracts

Many IBC connected blockchains utilise smart contracts, these could be written in rust, solidity or javascript depending on the smart contract platform being used.  Through the localhost client, a user can interact with and call into smart contracts by sending ibc messages from the localhost client to the smart contract. 

On Agoric, a planned use case is for managing delegations and staking with smart contracts on a given chain. MsgDelegate, MsgUndelegate, MsgBeginRedelegate messages would be sent using the localhost client to the staking module. The same interface could be used to manage staking cross chain, sending the same messages through IBC to an interchain account.

### Interoperability Infrastructure

Polymer plans to leverage the localhost client with multiple connections as part of their architecture to connect chains not using Tendermint or the Cosmos SDK to IBC enabled chains. Polymer will act as a hub connecting multiple blockchains together. 

# Functional requirements

## Assumptions and dependencies

1. Chains utilising the localhost client must be using an ibc-go release in the v7 line.
2. The channel behaviour of a localhost client will be as ICS 04 specifies.

## Features

### 1 - Configuration

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 1.01 | The localhost client shall have a client ID of the string `09-localhost`. | [Localhost client is created with client ID `09-localhost`](https://github.com/cosmos/ibc-go/blob/release/v7.1.x/modules/core/02-client/keeper/keeper.go#L60). | `Verified` | v7.1.0 |
| 1.02 | The localhost client shall have a sentinel connection ID of the string `connection-localhost`. | [Creation of sentinel connection with ID `connection-localhost`](https://github.com/cosmos/ibc-go/blob/release/v7.1.x/modules/core/03-connection/keeper/keeper.go#L200). | `Verified` | v7.1.0 | 
| 1.03 | Only 1 localhost connection is required | Localhost connection handshakes are forbidden in [Init](https://github.com/cosmos/ibc-go/blob/release/v7.1.x/modules/core/03-connection/types/msgs.go#L47) and [Try](https://github.com/cosmos/ibc-go/blob/release/v7.1.x/modules/core/03-connection/types/msgs.go#L110). | `Verified` | v7.1.0 |
| 1.04 | When the localhost client is initialised the consensus state must be `nil`. | [Error is returned if consensus state is not `nil`](https://github.com/cosmos/ibc-go/blob/release/v7.1.x/modules/light-clients/09-localhost/client_state.go#L57). | `Verified` | v7.1.0 |
| 1.05 | The localhost client can be added to a chain through an upgrade. | [Automatic migration handler configured in core IBC module to set the localhost `ClientState` and sentinel `ConnectionEnd` in state.](https://github.com/cosmos/ibc-go/blob/release/v7.1.x/modules/core/module.go#L132-L145). | `Verified` | v7.1.0 |
| 1.06 | A chain can enable the localhost client by initialising the client in the genesis state. | [`InitGenesis` handler in 02-client](https://github.com/cosmos/ibc-go/blob/release/v7.1.x/modules/core/02-client/genesis.go#L52) and [`InitGenesis` handler in 03-connection](https://github.com/cosmos/ibc-go/blob/release/v7.1.x/modules/core/03-connection/genesis.go#L23). | `Verified` | v7.1.0 |

### 2 - Operation

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 2.01 | A user of the localhost client can send IBC messages to an application on the same chain. | [e2e test with transfer](https://github.com/cosmos/ibc-go/blob/main/e2e/tests/transfer/localhost_test.go#L32). | `Verified`| v7.1.0 |
| 2.02 | A user can use the localhost client through the existing IBC application module interfaces. | [e2e test with transfer](https://github.com/cosmos/ibc-go/blob/main/e2e/tests/transfer/localhost_test.go#L32). | `Verified` | v7.1.0 | 

# External interface requirements

## 3 - CLI

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 3.01 | Existing CLI interfaces used with IBC application modules shall be useable with the localhost client. | Manual test | `Verified` | v7.1.0 |
