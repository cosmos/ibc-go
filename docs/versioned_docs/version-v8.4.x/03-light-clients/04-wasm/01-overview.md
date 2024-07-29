---
title: Overview
sidebar_label: Overview
sidebar_position: 1
slug: /ibc/light-clients/wasm/overview
---

# `08-wasm`

## Overview

Learn about the `08-wasm` light client proxy module. 

### Context

Traditionally, light clients used by ibc-go have been implemented only in Go, and since ibc-go v7 (with the release of the 02-client refactor), they are [first-class Cosmos SDK modules](/architecture/adr-010-light-clients-as-sdk-modules). This means that updating existing light client implementations or adding support for new light clients is a multi-step, time-consuming process involving on-chain governance: it is necessary to modify the codebase of ibc-go (if the light client is part of its codebase), re-build chains' binaries, pass a governance proposal and have validators upgrade their nodes. 

### Motivation

To break the limitation of being able to write light client implementations only in Go, the `08-wasm` adds support to run light clients written in a Wasm-compilable language. The light client byte code implements the entry points of a [CosmWasm](https://docs.cosmwasm.com/docs/) smart contract, and runs inside a Wasm VM. The `08-wasm` module exposes a proxy light client interface that routes incoming messages to the appropriate handler function, inside the Wasm VM, for execution.

Adding a new light client to a chain is just as simple as submitting a governance proposal with the message that stores the byte code of the light client contract. No coordinated upgrade is needed. When the governance proposal passes and the message is executed, the contract is ready to be instantiated upon receiving a relayer-submitted `MsgCreateClient`. The process of creating a Wasm light client is the same as with a regular light client implemented in Go.

### Use cases

- Development of light clients for non-Cosmos ecosystem chains: state machines in other ecosystems are, in many cases, implemented in Rust, and thus there are probably libraries used in their light client implementations for which there is no equivalent in Go. This makes the development of a light client in Go very difficult, but relatively simple to do it in Rust. Therefore, writing a CosmWasm smart contract in Rust that implements the light client algorithm becomes a lower effort.
