---
title: Client
sidebar_label: Client
sidebar_position: 7
slug: /ibc/light-clients/wasm/client
---

# Client

## CLI

A user can query and interact with the `08-wasm` module using the CLI. Use the `--help` flag to discover the available commands:

### Transactions

The `tx` commands allow users to interact with the `08-wasm` submodule.

```shell
simd tx ibc-wasm --help
```

#### `store-code`

TODO: document CLI to submit gov v1 proposal with `MsgStoreCode`.

### Query

The `query` commands allow users to query `08-wasm` state.

```shell
simd query ibc-wasm --help
```

#### `code-hashes`

The `code-hashes` command allows users to query the list of code hashes of Wasm light client contracts stored in the Wasm VM via the `MsgStoreCode`.

```shell
simd query ibc-wasm code-hashes [flags]
```

Example:

```shell
simd query ibc-wasm code-hashes
```

Example Output:

```shell
code_hashes:
- c64f75091a6195b036f472cd8c9f19a56780b9eac3c3de7ced0ec2e29e985b64
pagination:
  next_key: null
  total: "1"
```

#### `code`

The `code` command allows users to query the Wasm byte code of a light client contract given the provided input code hash.

```shell
./simd q ibc-wasm code
```

Example:

```shell
simd query ibc-wasm code c64f75091a6195b036f472cd8c9f19a56780b9eac3c3de7ced0ec2e29e985b64
```

Example Output:

```shell
code: AGFzb...AqBBE=
```

## gRPC

A user can query the `08-wasm` module using gRPC endpoints.

### `CodeHashes`

The `CodeHashes` endpoint allows users to query the list of code hashes of Wasm light client contracts stored in the Wasm VM via the `MsgStoreCode`.

```shell
ibc.lightclients.wasm.v1.Query/CodeHashes
```

Example:

```shell
grpcurl -plaintext \
  -d '{}' \
  localhost:9090 \
  ibc.lightclients.wasm.v1.Query/CodeHashes
```

Example output:

```shell
{
  "codeIds": [
    "c64f75091a6195b036f472cd8c9f19a56780b9eac3c3de7ced0ec2e29e985b64"
  ],
  "pagination": {
    "total": "1"
  }
}
```

### `Code`

The `Code` endpoint allows users to query the Wasm byte code of a light client contract given the provided input code hash.

```shell
ibc.lightclients.wasm.v1.Query/Code
```

Example:

```shell
grpcurl -plaintext \
  -d '{"code_hash":"c64f75091a6195b036f472cd8c9f19a56780b9eac3c3de7ced0ec2e29e985b64"}' \
  localhost:9090 \
  ibc.lightclients.wasm.v1.Query/Code
```

Example output:

```shell
{
  "code": AGFzb...AqBBE=
}
```
