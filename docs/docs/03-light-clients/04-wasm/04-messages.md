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
  // signer address
  Signer string
  // wasm byte code of light client contract. It can be raw or gzip compressed
  WasmByteCode []byte
}
```

This message is expected to fail if:

- `Signer` is an invalid Bech32 address, or it does not match the designated authority address.
- `WasmByteCode` is empty or it exceeds the maximum size, currently set to 3MB.

Only light client contracts stored using `MsgStoreCode` are allowed to be instantiated. An attempt to create a light client from contracts uploaded via other means (e.g. through `x/wasm` if the module shares the same Wasm VM instance with 08-wasm) will fail. Due to the idempotent nature of the Wasm VM's `StoreCode` function, it is possible to store the same byte code multiple times.

When execution of `MsgStoreCode` succeeds, the code hash of the contract (i.e. the sha256 hash of the contract's byte code) is stored in an allow list. When a relayer submits [`MsgCreateClient`](https://github.com/cosmos/ibc-go/blob/v7.2.0/proto/ibc/core/client/v1/tx.proto#L25-L37) with 08-wasm's `ClientState`, the client state includes the code hash of the Wasm byte code that should be called. Then 02-client calls [08-wasm's implementation of `Initialize` function](https://github.com/cosmos/ibc-go/blob/v7.2.0/modules/core/02-client/keeper/client.go#L34) (which is an interface function part of `ClientState`), and it will check that the code hash in the client state matches one of the coded hashes in the allow list. If a match is found, the light client is initialized; otherwise, the transaction is aborted.

## `MsgMigrateContract`

Migrating a contract to a new Wasm byte code is achieved by means of `MsgMigrateContract`:

```go
type MsgMigrateContract struct {
 	// signer address
  Signer string
 	// the client id of the contract
 	ClientId string
 	// the code hash of the new wasm byte code for the contract
 	CodeHash []byte
 	// the json-encoded migrate msg to be passed to the contract on migration
 	Msg []byte
}
```

This message is expected to fail if:

- `Signer` is an invalid Bech32 address, or it does not match the designated authority address.
- `ClientId` is not a valic identifier prefixed by `08-wasm`.
- `CodeHash` is not exactly 32 bytes long or it is not found in the list of allowed code hashes (a new code hash is added to the list when executing `MsgStoreCode`), or it matches the current code hash of the contract.

When a Wasm light client contract is migrated to a new Wasm byte code the code hash for the contract will be updated with the new code hash.

## `MsgRemoveCodeHash`

Removing a code hash from the allow list is achieved by means of `MsgRemoveCodeHash`:

```go
type MsgRemoveCodeHash struct {
 	// signer address
 	Signer string
 	// code hash to be removed from the store
 	CodeHash []byte
}
```

This message is expected to fail if:

- `Signer` is an invalid Bech32 address, or it does not match the designated authority address.
- `CodeHash` is not exactly 32 bytes long or it is not found in the list of allowed code hashes (a new code hash is added to the list when executing `MsgStoreCode`).

When a code hash is removed from the allow list, then the corresponding Wasm byte code will not be available for instantiation in [08-wasm's implementation of `Initialize` function](https://github.com/cosmos/ibc-go/blob/v7.2.0/modules/core/02-client/keeper/client.go#L34).

