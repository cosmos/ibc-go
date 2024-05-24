---
title: Concepts
sidebar_label: Concepts
sidebar_position: 2
slug: /ibc/light-clients/wasm/concepts
---

# Concepts

Learn about the differences between a proxy light client and a Wasm light client. 

## Proxy light client

The `08-wasm` module is not a regular light client in the same sense as, for example, the 07-tendermint light client. `08-wasm` is instead a *proxy* light client module, and this means that the module acts a proxy to the actual implementations of light clients. The module will act as a wrapper for the actual light clients uploaded as Wasm byte code and will delegate all operations to them (i.e. `08-wasm` just passes through the requests to the Wasm light clients). Still, the `08-wasm` module implements all the required interfaces necessary to integrate with core IBC, so that 02-client can call into it as it would for any other light client module. These interfaces are `LightClientModule`, `ClientState`, `ConsensusState` and `ClientMessage`, and we will describe them in the context of `08-wasm` in the following sections. For more information about this set of interfaces, please read section [Overview of the light client module developer guide](../01-developer-guide/01-overview.md#overview).

### `LightClientModule`

The `08-wasm`'s `LightClientModule` data structure contains two fields:

- `keeper` is the `08-wasm` module keeper.
- `storeProvider` encapsulates the IBC core store key and provides access to isolated prefix stores for each client so they can read/write in separate namespaces.

```go
type LightClientModule struct {
  keeper        wasmkeeper.Keeper
  storeProvider exported.ClientStoreProvider
}
```

See section [`LightClientModule` of the light client module developer guide](../01-developer-guide/01-overview.md#lightclientmodule) for more information about the `LightClientModule` interface.

### `ClientState`

The `08-wasm`'s `ClientState` data structure contains three fields:

- `Data` contains the bytes of the Protobuf-encoded client state of the underlying light client implemented as a Wasm contract. For example, if the Wasm light client contract implements the GRANDPA light client algorithm, then `Data` will contain the bytes for a [GRANDPA client state](https://github.com/ComposableFi/composable-ibc/blob/02ce69e2843e7986febdcf795f69a757ce569272/light-clients/ics10-grandpa/src/proto/grandpa.proto#L35-L60).
- `Checksum` is the sha256 hash of the Wasm contract's byte code. This hash is used as an identifier to call the right contract.
- `LatestHeight` is the latest height of the counterparty state machine (i.e. the height of the blockchain), whose consensus state the light client tracks.

```go
type ClientState struct {
  // bytes encoding the client state of the underlying 
  // light client implemented as a Wasm contract
  Data         []byte
  // sha256 hash of Wasm contract byte code
  Checksum     []byte
  // latest height of the counterparty ledger
  LatestHeight types.Height
}
```

See section [`ClientState` of the light client module developer guide](../01-developer-guide/01-overview.md#clientstate) for more information about the `ClientState` interface.

### `ConsensusState`

The `08-wasm`'s `ConsensusState` data structure maintains one field:

- `Data` contains the bytes of the Protobuf-encoded consensus state of the underlying light client implemented as a Wasm contract. For example, if the Wasm light client contract implements the GRANDPA light client algorithm, then `Data` will contain the bytes for a [GRANDPA consensus state](https://github.com/ComposableFi/composable-ibc/blob/02ce69e2843e7986febdcf795f69a757ce569272/light-clients/ics10-grandpa/src/proto/grandpa.proto#L87-L94).

```go
type ConsensusState struct {
  // bytes encoding the consensus state of the underlying light client
  // implemented as a Wasm contract.
  Data []byte
}
```

See section [`ConsensusState` of the light client module developer guide](../01-developer-guide/01-overview.md#consensusstate) for more information about the `ConsensusState` interface.

### `ClientMessage`

`ClientMessage` is used for performing updates to a `ClientState` stored on chain. The `08-wasm`'s `ClientMessage` data structure maintains one field:

- `Data` contains the bytes of the Protobuf-encoded header(s) or misbehaviour for the underlying light client implemented as a Wasm contract. For example, if the Wasm light client  contract implements the GRANDPA light client algorithm, then `Data` will contain the bytes of either [header](https://github.com/ComposableFi/composable-ibc/blob/02ce69e2843e7986febdcf795f69a757ce569272/light-clients/ics10-grandpa/src/proto/grandpa.proto#L96-L104) or [misbehaviour](https://github.com/ComposableFi/composable-ibc/blob/02ce69e2843e7986febdcf795f69a757ce569272/light-clients/ics10-grandpa/src/proto/grandpa.proto#L106-L112) for a GRANDPA light client.

```go
type ClientMessage struct {
  // bytes encoding the header(s) or misbehaviour for the underlying light client
  // implemented as a Wasm contract.
  Data []byte
}
```

See section [`ClientMessage` of the light client module developer guide](../01-developer-guide/01-overview.md#clientmessage) for more information about the `ClientMessage` interface.

## Wasm light client

The actual light client can be implemented in any language that compiles to Wasm and implements the interfaces of a [CosmWasm](https://docs.cosmwasm.com/docs/) contract. Even though in theory other languages could be used, in practice (at least for the time being) the most suitable language to use would be Rust, since there is already good support for it for developing CosmWasm smart contracts.

At the moment of writing there are two contracts available: one for [Tendermint](https://github.com/ComposableFi/composable-ibc/tree/master/light-clients/ics07-tendermint-cw) and one [GRANDPA](https://github.com/ComposableFi/composable-ibc/tree/master/light-clients/ics10-grandpa-cw) (which is being used in production in [Composable Finance's Centauri bridge](https://github.com/ComposableFi/composable-ibc)). And there are others in development (e.g. for Near).
