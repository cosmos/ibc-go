<!--
order: 9
-->

# Client

## CLI

A user can query and interact with the `transfer` module using the CLI. Use the `--help` flag to discover the available commands:

### Query

The `query` commands allow users to query `transfer` state.

```shell
simd query ibc-transfer --help
```

#### `total-escrow`

The `total-escrow` command allows users to query the total amount in escrow for a particular coin denomination regardless of the transfer channel from where the coins were sent out.

```shell
simd query ibc-transfer total-escrow [denom] [flags]
```

Example:

```shell
simd query ibc-transfer total-escrow samoleans
```

Example Output:

```shell
amount: "100"
```

## gRPC

A user can query the `transfer` module using gRPC endpoints.

### `TotalEscrowForDenom`

The `TotalEscrowForDenom` endpoint allows users to query the total amount in escrow for a particular coin denomination regardless of the transfer channel from where the coins were sent out.

```shell
ibc.applications.transfer.v1.Query/TotalEscrowForDenom
```

Example:

```shell
grpcurl -plaintext \
  -d '{"denom":"samoleans"}' \
  localhost:9090 \
  ibc.applications.transfer.v1.Query/TotalEscrowForDenom
```

Example output:

```shell
{
  "amount": "100"
}
```