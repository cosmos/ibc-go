# Business requirements

Using IBC as a mean of communicating between chains and ecosystems has proven to be useful within Cosmos. There is then value in extending
this feature into other ecosystems, bringing a battle-tested protocol of trusted communication as an option to send assets and data.

This is especially useful to protocols and companies whose business model is to improve cross-chain user interface, or to enable it when
it's not. The main use case for this is bridging assets between chains. There are multiple protocols and companies currently performing such
a service but none has yet been able to do it using IBC outside of the Cosmos ecosystem.

A core piece for this to happen is to have a light client implementation of each ecosystem that has to be integrated, and uses a **new** consensus
algorithm. This module broadens the horizon of light client development to not be limited to using Golang only for chains wanting to use IBC and ibc-go,
but instead expands the choice to any programming language and toolchain that is able to compile to Wasm instead.

Bridging assets is likely the simplest form of interchain communication. Its value is confirmed on a daily basis, when considering the volumes that protocols
like [Axelar](https://dappradar.com/multichain/defi/axelar-network), Gravity, [Wormhole](https://dappradar.com/multichain/defi/wormhole/) and
Layer0 process.

## Problem

In order to export IBC outside of Tendermint-based ecosystems, there is a need to introduce new light clients. This is a core need for
companies and protocols trying to bridge ecosystems such as Ethereum, NEAR, Polkadot, etc. as none of these uses Tendermint as their
consensus mechanism. Introducing a new light client implementation is not straightforward. The implementor needs to follow the light client's
specification, and will try to make use of all available tools to keep the development cost reasonable.

Normally, most of available tools to implement a light client stem from the blockchain ecosystem this client belongs to. Say for example, if a developer
wants to implement the Polkadot finality gadget called GRANDPA, she will find that most of the tools are available on Substrate. Hence, being able to have a way
to let developers implement these light clients using the best and most accessible tools for the job is very beneficial, as it avoids having to re-implement
features that are otherwise available and likely heavily audited already. And since Wasm is a well supported target that most programming languages support,
it becomes a proper solution to port the code for ibc-go to interpret without requiring the entire light client being written using Go. 

## Objectives

The objective of this module is to have allow two chains with heterogenous consensus algorithms being connected through light clients that are not necesarily written in Go, but compiled to Wasm instead.

## Scope

The scope of this feature is to allow any implemention written in Wasm to be compliant with the interface 
expressed in [02-client `ClientState` interface](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L44-L139).

| Features               | Release |
| ---------------------- | ------- |
| Dispatch messages to a light client written in Wasm following the `ClientState` interface. | v1 |
| Support GRANDPA light client. | v1 |

# User requirements

## Use cases

The first use case that this module will enable is the connection between GRANDPA light client chains and Tendermint light client chains. Further implementation of other light clients, such as NEAR, Ethereum, etc. will likely consider building on top of this module.

# Functional requirements

## Assumptions

1. This feature expects the [02-client refactor completed](https://github.com/cosmos/ibc-go/milestone/16), which is enabled in ibc-go v7.

## Features

### 1 - Configuration

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ | 
| 1.01 | To enable the usage of the Wasm client module, chains must add update the `AllowedClients` parameter in the 02-client submodule. | TBD | Drafted |
| 1.02 | The genesis state of the Wasm client module consists of the code ID and bytecode for each light client Wasm contract. | TBD | `Drafted` |
| 1.03 | A chain shall have the ability to export the Wasm client module genesis state.	| TBD | `Drafted` |
| 1.04 | A chain shall have the ability to initialize the Wasm client module genesis state. | TBD | `Drafted` |

### 2 - Initiation

| ID | Description | Verification | Status | 
| -- | ----------- | ------------ | ------ |
| 2.01 | Users must submit a governance proposal to store a light client implementation compiled in Wasm bytecode. | TBD | `Drafted` |
| 2.02 | Once a light client Wasm contract has been stored, every light client will be created with a new instance of the contract. | TBD | `Drafted` |
| 2.03 | The bytecode for each light client Wasm contract is stored in a client-prefixed store indexed by the hash of the bytecode. | TBD | `Drafted` |
| 2.04 | The size in bytes of bytecode of the light client Wasm contract must be > 0 and <= 3 MiB. | TBD | `Drafted` |

# Non-functional requirements

## 3 - Memory 

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 3.01 | Each contract execution memory limit is 4096 MiB. | TBD | `Drafted` | 

# External interface requirements

## 4 - CLI

### Query

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 4.01 | There shall be a CLI command available to query the bytecode of a light client Wasm contract by code ID. | TBD | `Drafted` |
| 4.02 | There shall be a CLI command available to query the code IDs for all deployed light client Wasm contracts. | TBD | `Drafted` |