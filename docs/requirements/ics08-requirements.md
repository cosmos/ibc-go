<!-- markdown-link-check-disable -->
# Business requirements

Using IBC as a mean of communicating between chains and ecosystems has proven to be useful within Cosmos. There is then value in extending
this feature into other ecosystems, bringing a battle-tested protocol of trust-minimized communication as an option to send assets and data.

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
consensus mechanism. Introducing a new light client implementation is not straightforward: sometimes cryptographic primitives are not
available, or support for operating with certain data structures (like specific tries/trees, etc) are not available in Go. The implementer needs to follow 
the light client's specification, and will try to make use of all available tools to keep the development cost reasonable.

Normally, most of available tools to implement a light client stem from the blockchain ecosystem this client belongs to. Say for example, if a developer
wants to implement the Polkadot finality gadget called GRANDPA, she will find that most of the tools are available on Substrate. Hence, being able to have a way
to let developers implement these light clients using the best and most accessible tools for the job is very beneficial, as it avoids having to re-implement
features that are otherwise available and likely heavily audited already. And since Wasm is a well supported target that most programming languages support,
it becomes a proper solution to port the code for ibc-go to interpret without requiring the entire light client being written using Go. 

## Objectives

The objective of this module is to have allow two chains with heterogeneous consensus algorithms being connected through light clients that are not necessarily written in Go, but compiled to Wasm instead.

## Scope

The scope of this feature is to allow any implementation written in Wasm to be compliant with the interface 
expressed in [02-client `ClientState` interface](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L44-L139).

| Features               | Release |
| ---------------------- | ------- |
| Store light client contract bytecode by means of a governance proposal. | v1 |
| Dispatch messages to a light client written in Wasm following the `ClientState` interface. | v1 |
| Migrate the contract instance of a light client to a newer contract bytecode. | v1 |
| Remove checksums from the list of allowed checksums to disallow contract instantiation. | v1 |
| Support GRANDPA light client. | v1 |

# User requirements

## Use cases

The first use case that this module will enable is the connection between GRANDPA light client chains and Tendermint light client chains. Further implementation of other light clients, such as NEAR, Ethereum, etc. will likely consider building on top of this module.

# Functional requirements

## Assumptions

1. This feature expects the [02-client refactor completed](https://github.com/cosmos/ibc-go/milestone/16), which is enabled in ibc-go v7.

## Features

### 1 - Configuration

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 1.01 | To enable the usage of the Wasm client module, chains must update the `AllowedClients` parameter in the 02-client submodule. | The `AllowedClients` needs to be updated to add the `08-wasm` client type.  | `Verified` | v0.1.0 |
| 1.02 | The genesis state of the Wasm client module consists of the list of contracts' bytecode for each light client Wasm contract. | See [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/proto/ibc/lightclients/wasm/v1/genesis.proto#L12). | `Verified` | v0.1.0 |
| 1.03 | A chain shall have the ability to initialize the Wasm client module genesis state. | See [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/keeper/genesis.go#L12). | `Verified` | v0.1.0 |
| 1.04 | A chain shall have the ability to export the Wasm client module genesis state.	| See [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/keeper/genesis.go#L24). | `Verified` | v0.1.0 |
| 1.05 | Chains that integrate the wasmd module may have the option to use the same wasm VM instance for both wasmd and the 08-wasm module. | A [keeper constructor function](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/keeper/keeper.go#L39) is provided that accepts a wasm VM pointer. | `Verified` | v0.1.0 |
| 1.06 | Chains that do not integrate the wasmd module may have the option to delegate to the 08-wasm module the instantiation of the necessary wasm VM. | A [keeper constructor function](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/keeper/keeper.go#L88) is provided that accepts parameters to configure the wasm VM that would instantiated by the module. | `Verified` | v0.1.0 |
| 1.07 | It may be possible to register custom query plugins for the 08-wasm module. | See [parameter in keeper constructor function](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/keeper/keeper.go#L45). | `Verified` | v0.1.0 |

### 2 - Initiation

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 2.01 | Users must submit a governance proposal to store a light client implementation compiled in Wasm bytecode. | [`MsgStoreCode` is authority-gated](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/keeper/msg_server.go#L20). | `Verified` | v0.1.0 |
| 2.02 | Once a light client Wasm contract has been stored, every light client will be created with a new instance of the contract. | The [`Instantiate` endpoint](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/types/client_state.go#L129) of the contract is called when creating a new light client. | `Verified` | v0.1.0 |
| 2.03 | It must not be possible to initialize a light client with a bytecode checksum that has not been previously stored via `MsgStoreCode`. | Se check [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/types/client_state.go#L119). | `Verified` | v0.1.0 |

### 3 - Contract migration

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 3.01 | Users may submit a governance proposal to remove a particular bytecode checksum from the list of allowed checksums. | [`MsgRemoveChecksum`](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/proto/ibc/lightclients/wasm/v1/tx.proto#L39) is available. | `Verified` | v0.1.0 |
| 3.02 | Users may submit a governance proposal to migrate a light client to a new contract instance specified by its contract checksum. | [`MsgMigrateContract`](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/proto/ibc/lightclients/wasm/v1/tx.proto#L52) is available. | `Verified` | v0.1.0 |

# Non-functional requirements

## 4 - Storage

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 4.01 | The bytecode for each light client Wasm contract does not need to be stored in a client-prefixed store. | The [bytecode is stored only in the wasm VM](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/keeper/keeper.go#L137). | `Verified` | v0.1.0 |
| 4.02 | When a contract bytecode is stored it should also be pinned the wasm VM in-memory cache. | See [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/keeper/keeper.go#L148). | `Verified` | v0.1.0 |
| 4.03 | The size in bytes of bytecode of the light client Wasm contract must be > 0 and <= 3 MiB. | See validation [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/types/validation.go#L20). | `Verified` | v0.1.0 |

## 5 - Memory 

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 5.01 | Each contract execution memory limit is 32 MiB. | See [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/types/config.go#L8). | `Verified` | v0.1.0 | 

## 6 - Security 

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 6.01 | The 08-wasm module must ensure that the contracts do not remove or corrupt the stored client state state. | See [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/types/vm.go#L227). | `Verified` | v0.1.0 | 
| 6.02 | The 08-wasm module must ensure that the contracts do not include in the response to sudo, instantiate or migrate calls messages, events or attributes. | See [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/types/vm.go#L300). |  `Verified` | v0.1.0 | 

# External interface requirements

## 7 - CLI

### Query

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 6.01 | There shall be a CLI command available to query the bytecode of a light client Wasm contract by checksum. | See [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/keeper/grpc_query.go#L23). | `Verified` | v0.1.0 |
| 7.02 | There shall be a CLI command available to query the checksums for all deployed light client Wasm contracts. | See [here](https://github.com/cosmos/ibc-go/blob/modules/light-clients/08-wasm/v0.1.0%2Bibc-go-v8.0-wasmvm-v1.5/modules/light-clients/08-wasm/keeper/grpc_query.go#L49). | `Verified` | v0.1.0 |
<!-- markdown-link-check-enable-->
