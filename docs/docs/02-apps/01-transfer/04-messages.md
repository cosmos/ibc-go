---
title: Messages
sidebar_label: Messages
sidebar_position: 4
slug: /apps/transfer/messages
---

# Messages

## `MsgTransfer`

A fungible token cross chain transfer is achieved by using the `MsgTransfer`:

```go
type MsgTransfer struct {
  SourcePort        string
  SourceChannel     string
  // Deprecated: Use Tokens instead.
  Token             sdk.Coin
  Sender            string
  Receiver          string
  TimeoutHeight     ibcexported.Height
  TimeoutTimestamp  uint64
  Memo              string
  Tokens            []sdk.Coin
  Forwarding        *Forwarding
}

type Forwarding struct {
	Unwind bool
	Hops  []Hop
}

type Hop struct {
	PortId    string
	ChannelId string
}
```

:::info
Multi-denom token transfers and token forwarding are features supported only on ICS20 v2 transfer channels.
:::

If `Forwarding` is `nil`, this message is expected to fail if:

- `SourcePort` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators).
- `SourceChannel` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).
- `Tokens` must not be empty.
- Each `Coin` in `Tokens` must satisfy the following:
    - `Amount` must be positive.
    - `Denom` must be a valid IBC denomination, as defined in [ADR 001 - Coin Source Tracing](/architecture/adr-001-coin-source-tracing).
- `Sender` is empty.
- `Receiver` is empty or contains more than 2048 bytes.
- `Memo` contains more than 32768 bytes.
- `TimeoutHeight` and `TimeoutTimestamp` are both zero.

If `Forwarding` is not `nil`, then to use forwarding you must either set `Unwind` to true or provide a non-empty list of `Hops`. Setting both `Unwind` to true and providing a non-empty list of `Hops` is allowed, but the total number of hops that is formed as a combination of the hops needed to unwind the tokens and the hops to forward them afterwards to the final destination must not exceed 8. When using forwarding, timeout must be specified using only `TimeoutTimestamp` (i.e. `TimeoutHeight` must be zero). Please note that the timeout timestamp must take into account the time that it may take tokens to be forwarded through the intermediary chains. Additionally, please note that the `MsgTransfer` will fail if:

- `Hops` is not empty, and the number of elements of `Hops` is greater than 8, or either the `PortId` or `ChannelId` of any of the `Hops` is not a valid identifier.
- `Unwind` is true, and either the coins to be transferred have different denomination traces, or `SourcePort` and `SourceChannel` are not empty strings (they must be empty because they are set by the transfer module, since it has access to the denomination trace information and is thus able to know the source port ID, channel ID to use in order to unwind the tokens). If `Unwind` is true, the transfer module expects the tokens in `MsgTransfer` to not be native to the sending chain (i.e. they must be IBC vouchers).

Please note that the `Token` field is deprecated and users should now use `Tokens` instead. If `Token` is used then `Tokens` must be empty. Similarly, if `Tokens` is used then `Token` should be left empty. This message will send a fungible token to the counterparty chain represented by the counterparty Channel End connected to the Channel End with the identifiers `SourcePort` and `SourceChannel`.

The denomination provided for transfer should correspond to the same denomination represented on this chain. The prefixes will be added as necessary upon by the receiving chain.

If the `Amount` is set to the maximum value for a 256-bit unsigned integer (i.e. 2^256 - 1), then the whole balance of the corresponding denomination will be transferred. The helper function `UnboundedSpendLimit` in the `types` package of the `transfer` module provides the sentinel value that can be used.

### Memo

The memo field was added to allow applications and users to attach metadata to transfer packets. The field is optional and may be left empty. When it is used to attach metadata for a particular middleware, the memo field should be represented as a json object where different middlewares use different json keys.

For example, the following memo field is used by the [callbacks middleware](../../04-middleware/02-callbacks/01-overview.md) to attach a source callback to a transfer packet:

```jsonc
{
  "src_callback": {
    "address": "callbackAddressString",
    // optional
    "gas_limit": "userDefinedGasLimitString",
  }
}
```

You can find more information about other applications that use the memo field in the [chain registry](https://github.com/cosmos/chain-registry/blob/master/_memo_keys/ICS20_memo_keys.json).

Please note that the memo field is always meant to be consumed only on the final destination chain. This means that the transfer module will guarantee that the memo field in the intermediary chains is empty.
