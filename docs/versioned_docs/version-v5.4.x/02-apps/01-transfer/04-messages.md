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
  Token             sdk.Coin
  Sender            string
  Receiver          string
  TimeoutHeight     ibcexported.Height
  TimeoutTimestamp  uint64
  Memo              string
}
```

This message is expected to fail if:

- `SourcePort` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators).
- `SourceChannel` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).
- `Token` is invalid (denom is invalid or amount is negative)
    - `Token.Amount` is not positive.
    - `Token.Denom` is not a valid IBC denomination as per [ADR 001 - Coin Source Tracing](/architecture/adr-001-coin-source-tracing).
- `Sender` is empty.
- `Receiver` is empty.
- `TimeoutHeight` and `TimeoutTimestamp` are both zero.

This message will send a fungible token to the counterparty chain represented by the counterparty Channel End connected to the Channel End with the identifiers `SourcePort` and `SourceChannel`.

The denomination provided for transfer should correspond to the same denomination represented on this chain. The prefixes will be added as necessary upon by the receiving chain.

### Memo

The memo field was added to allow applications and users to attach metadata to transfer packets. The field is optional and may be left empty. When it is used to attach metadata for a particular middleware, the memo field should be represented as a json object where different middlewares use different json keys.

You can find more information about applications that use the memo field in the [chain registry](https://github.com/cosmos/chain-registry/blob/master/_memo_keys/ICS20_memo_keys.json).
