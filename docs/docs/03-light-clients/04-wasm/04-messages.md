---
title: Messages
sidebar_label: Messages
sidebar_position: 4
slug: /ibc/light-clients/wasm/messages
---

# Messages

## `MsgStoreCode`

Uploading the Wasm light client contract to the Wasm VM storage is achieved by means of `MsgStoreCode`:

```go
type MsgStoreCode struct {
  Signer string
  // wasm byte code of light client contract. It can be raw or gzip compressed
  WasmByteCode []byte
}
```

This message is expected to fail if:

- `Signer` is an invalid Bech32 address, or it does not match the designated authority address.
- `WasmByteCode` is empty or it exceeds the maximum size, currently set to 3MB.

Only light client contracts stored using `MsgStoreCode` are allowed to be instantiated. An attempt to create a light client from contracts uploaded via other means (e.g. through `x/wasm` if the module shares the same Wasm VM instance with 08-wasm) will fail. Due to the idempotent nature of the Wasm VM's `StoreCode` function, it is possible to store the same bytecode multiple times.

When execution of `MsgStoreCode` succeeds, the code hash of the contract (i.e. the sha256 hash of the contract's bytecode) is stored in a list. When a relayer submits [`MsgCreateClient`](https://github.com/cosmos/ibc-go/blob/v7.2.0/proto/ibc/core/client/v1/tx.proto#L25-L37) with 08-wasm's `ClientState`, the client state includes the code hash of the contract that should be called. Then 02-client calls [08-wasm's implementation of `Initialize` function](https://github.com/cosmos/ibc-go/blob/v7.2.0/modules/core/02-client/keeper/client.go#L34) (which is an interface function part of `ClientState`), and it will check that the code hash in the client state matches one of the coded hashes in the list. If a match is found, the light client is initialized; otherwise, the transaction is aborted.
