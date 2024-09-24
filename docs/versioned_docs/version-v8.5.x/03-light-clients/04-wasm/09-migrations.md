---
title: Migrations
sidebar_label: Migrations
sidebar_position: 9
slug: /ibc/light-clients/wasm/migrations
---

# Migrations

This guide provides instructions for migrating 08-wasm versions.

## From ibc-go v7.3.x to ibc-go v8.0.x

## Chains

In the 08-wasm versions compatible with ibc-go v7.3.x and above from the v7 release line, the checksums of the uploaded Wasm bytecodes are all stored under a single key. From ibc-go v8.0.x the checksums are stored using [`collections.KeySet`](https://docs.cosmos.network/v0.50/build/packages/collections#keyset), whose full functionality became available in Cosmos SDK v0.50. There is therefore an [automatic migration handler](https://github.com/cosmos/ibc-go/blob/57fcdb9a9a9db9b206f7df2f955866dc4e10fef4/modules/light-clients/08-wasm/module.go#L115-L118) configured in the 08-wasm module to migrate the stored checksums to `collections.KeySet`.

## From v0.1.0+ibc-go-v8.0-wasmvm-v1.5 to v0.2.0-ibc-go-v8.3-wasmvm-v2.0

The `WasmEngine` interface has been updated to reflect changes in the function signatures of Wasm VM:

```diff
type WasmEngine interface {
- StoreCode(code wasmvm.WasmCode) (wasmvm.Checksum, error)
+ StoreCode(code wasmvm.WasmCode, gasLimit uint64) (wasmvmtypes.Checksum, uint64, error)

  StoreCodeUnchecked(code wasmvm.WasmCode) (wasmvm.Checksum, error)

  Instantiate(
    checksum wasmvm.Checksum,
    env wasmvmtypes.Env,
    info wasmvmtypes.MessageInfo,
    initMsg []byte,
    store wasmvm.KVStore,
    goapi wasmvm.GoAPI,
    querier wasmvm.Querier,
    gasMeter wasmvm.GasMeter,
    gasLimit uint64,
    deserCost wasmvmtypes.UFraction,
- ) (*wasmvmtypes.Response, uint64, error)
+ ) (*wasmvmtypes.ContractResult, uint64, error)

  Query(
    checksum wasmvm.Checksum,
    env wasmvmtypes.Env,
    queryMsg []byte,
    store wasmvm.KVStore,
    goapi wasmvm.GoAPI,
    querier wasmvm.Querier,
    gasMeter wasmvm.GasMeter,
    gasLimit uint64,
    deserCost wasmvmtypes.UFraction,
- ) ([]byte, uint64, error)
+ ) (*wasmvmtypes.QueryResult, uint64, error)

  Migrate(
    checksum wasmvm.Checksum,
    env wasmvmtypes.Env,
    migrateMsg []byte,
    store wasmvm.KVStore,
    goapi wasmvm.GoAPI,
    querier wasmvm.Querier,
    gasMeter wasmvm.GasMeter,
    gasLimit uint64,
    deserCost wasmvmtypes.UFraction,
- ) (*wasmvmtypes.Response, uint64, error)
+ ) (*wasmvmtypes.ContractResult, uint64, error)

  Sudo(
    checksum wasmvm.Checksum,
    env wasmvmtypes.Env,
    sudoMsg []byte,
    store wasmvm.KVStore,
    goapi wasmvm.GoAPI,
    querier wasmvm.Querier,
    gasMeter wasmvm.GasMeter,
    gasLimit uint64,
    deserCost wasmvmtypes.UFraction,
- ) (*wasmvmtypes.Response, uint64, error)
+ ) (*wasmvmtypes.ContractResult, uint64, error)

  GetCode(checksum wasmvm.Checksum) (wasmvm.WasmCode, error)

  Pin(checksum wasmvm.Checksum) error

  Unpin(checksum wasmvm.Checksum) error
}
```

Similar changes were required in the functions of the `MockWasmEngine` interface.

### Chains

The `SupportedCapabilities` field of `WasmConfig` is now of type `[]string`:

```diff
type WasmConfig struct {
  DataDir string
- SupportedCapabilities string
+ SupportedCapabilities []string
  ContractDebugMode bool
}
```
