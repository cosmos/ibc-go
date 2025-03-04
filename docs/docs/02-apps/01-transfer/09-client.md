---
title: Client
sidebar_label: Client
sidebar_position: 9
slug: /apps/transfer/client
---

# Client

## CLI

A user can query and interact with the `transfer` module using the CLI. Use the `--help` flag to discover the available commands:

### Query

The `query` commands allow users to query `transfer` state.

```shell
simd query ibc-transfer --help
```

#### Transactions

The `tx` commands allow users to interact with the controller submodule.

```shell
simd tx ibc-transfer --help
```

#### `transfer`

The `transfer` command allows users to execute cross-chain token transfers from the source port ID and channel ID on the sending chain.

```shell
simd tx ibc-transfer transfer [src-port] [src-channel] [receiver] [coins] [flags]
```

Multiple tokens can be transferred on the same transaction by specifying a comma-separated list 
of amount and denomination (e.g. `100uatom,100uosmo`) in the `coins` option.

The additional flags that can be used with the command are:

- `--packet-timeout-height` to specify the timeout block height in the format `{revision}-{height}`. The default value is `0-0`, which effectively disables the timeout. Timeout height can only be absolute, therefore this option must be used in combination with `--absolute-timeouts` set to true.
- `--packet-timeout-timestamp` to specify the timeout timestamp in nanoseconds. The timeout can be either relative (from the current UTC time) or absolute. The default value is 10 minutes (and thus relative).
- `--absolute-timeouts` to interpret the timeout timestamp as an absolute value (when set to true). The default value is false (and thus the timeout is considered relative to current UTC time).
- `--memo` to specify the memo string to be sent along with the transfer packet. If forwarding is used, then the memo string will be carried through the intermediary chains to the final destination.
- `--forwarding` to specify forwarding information in the form of a comma separated list of source port ID/channel ID pairs at each intermediary chain (e.g. `transfer/channel-0,transfer/channel-1`).
- `--unwind` to specify if the tokens must be automatically unwound to there origin chain. This option can be used in combination with `--forwarding` to forward the tokens to the final destination after unwinding. When this flag is true, the tokens specified in the `coins` option must all have the same denomination trace path (i.e. all tokens must be IBC vouchers sharing exactly the same set of destination port/channel IDs in their denomination trace path). Arguments `[src-port]` and  `[src-channel]` must not be passed if the `--unwind` flag is specified.

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
