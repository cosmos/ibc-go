<!--
order: 7
-->

# Client

## CLI

A user can query and interact with the `08-wasm` module using the CLI. Use the `--help` flag to discover the available commands:

### Transactions

The `tx` commands allow users to interact with the `08-wasm` submodule.

```shell
simd tx ibc-wasm --help
```

##### `store-code`

TODO: document CLI to submit gov v1 proposal with `MsgStoreCode`.

### Query

The `query` commands allow users to query `08-wasm` state.

```shell
simd query ibc-wasm --help
```

#### `code-hashes`

The `code-hashes` command allows users to query the list of code hashes of Wasm light client contracts in the Wasm VM via the `MsgStoreCode`.

```shell
simd query ibc-wasm code-hashes [flags]
```

Example:

```shell
simd query ibc-wasm code-hashes
```

Example Output:

```shell
amount: "100"
```

#### `code`

The `code` command allows users to query the list of code hashes of Wasm light client contracts in the Wasm VM via the `MsgStoreCode`.

```shell
./simd q ibc-wasm code
```

Example:

```shell
simd query ibc-wasm code <TODO: code-hash>
```

Example Output:

```shell
amount: "100"
```

## gRPC

A user can query the `08-wasm` module using gRPC endpoints.

### `CodeHashes`

The `CodeHashes` endpoint allows users to query the total amount in escrow for a particular coin denomination regardless of the transfer channel from where the coins were sent out.

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
  "amount": "100"
}
```

### `Code`

```shell
ibc.lightclients.wasm.v1.Query/Code
```
