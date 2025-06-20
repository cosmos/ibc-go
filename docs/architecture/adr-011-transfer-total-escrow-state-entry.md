# ADR 011: ICS-20 transfer state entry for total amount of tokens in escrow

## Changelog

- 2023-05-24: Initial draft

## Status

Accepted and applied in v7.1 of ibc-go

## Context

Every ICS-20 transfer channel has its own escrow bank account. This account is used to lock tokens that are transferred out of a chain that acts as the source of the tokens (i.e. when the tokens being transferred have not returned to the originating chain). This design makes it easy to query the balance of the escrow accounts and find out the total amount of tokens in escrow in a particular channel. However, there are use cases where it would be useful to determine the total escrowed amount of a given denomination across all channels where those tokens have been transferred out.

For example: assuming that there are three channels between Cosmos Hub to Osmosis and 10 ATOM have been transferred from the Cosmos Hub to Osmosis on each of those channels, then we would like to know that 30 ATOM have been transferred (i.e. are locked in the escrow accounts of each channel) without needing to iterate over each escrow account to add up the balances of each.

For a sample use case where this feature would be useful, please refer to Osmosis' rate limiting use case described in [#2664](https://github.com/cosmos/ibc-go/issues/2664).

## Decision

### State entry denom -> amount

The total amount of tokens in escrow (across all transfer channels) for a given denomination is stored in state in an entry keyed by the denomination: `totalEscrowForDenom/{denom}`.

### Panic if amount is negative

If a negative amount is ever attempted to be stored, then the keeper function will panic:

```go
if coin.Amount.IsNegative() {
  panic(fmt.Sprintf("amount cannot be negative: %s", coin.Amount))
}
```

### Delete state entry if amount is zero

When setting the amount for a particular denomination, the value might be zero if all tokens that were transferred out of the chain have been transferred back. If this happens, then the state entry for this particular denomination will be deleted, since Cosmos SDK's `x/bank` module prunes any non-zero balances:

```go
if coin.Amount.IsZero() {
  store.Delete(key) // delete the key since Cosmos SDK x/bank module will prune any non-zero balances
  return
}
```

### Bundle escrow/unescrow with setting state entry

Two new functions are implemented that bundle together the operations of escrowing/unescrowing and setting the total escrow amount in state, since these operations need to be executed together. 

For escrowing tokens:

```go
// escrowToken will send the given token from the provided sender to the escrow address. It will also
// update the total escrowed amount by adding the escrowed token to the current total escrow.
func (k Keeper) escrowToken(ctx sdk.Context, sender, escrowAddress sdk.AccAddress, token sdk.Coin) error {
  if err := k.bankKeeper.SendCoins(ctx, sender, escrowAddress, sdk.NewCoins(token)); err != nil {
    // failure is expected for insufficient balances
    return err
  }

  // track the total amount in escrow keyed by denomination to allow for efficient iteration
  currentTotalEscrow := k.GetTotalEscrowForDenom(ctx, token.GetDenom())
  newTotalEscrow := currentTotalEscrow.Add(token)
  k.SetTotalEscrowForDenom(ctx, newTotalEscrow)

  return nil
}
```

For unescrowing tokens:

```go
// unescrowToken will send the given token from the escrow address to the provided receiver. It will also
// update the total escrow by deducting the unescrowed token from the current total escrow.
func (k Keeper) unescrowToken(ctx sdk.Context, escrowAddress, receiver sdk.AccAddress, token sdk.Coin) error {
  if err := k.bankKeeper.SendCoins(ctx, escrowAddress, receiver, sdk.NewCoins(token)); err != nil {
    // NOTE: this error is only expected to occur given an unexpected bug or a malicious
    // counterparty module. The bug may occur in bank or any part of the code that allows
    // the escrow address to be drained. A malicious counterparty module could drain the
    // escrow address by allowing more tokens to be sent back then were escrowed.
    return errorsmod.Wrap(err, "unable to unescrow tokens, this may be caused by a malicious counterparty module or a bug: please open an issue on counterparty module")
  }

  // track the total amount in escrow keyed by denomination to allow for efficient iteration
  currentTotalEscrow := k.GetTotalEscrowForDenom(ctx, token.GetDenom())
  newTotalEscrow := currentTotalEscrow.Sub(token)
  k.SetTotalEscrowForDenom(ctx, newTotalEscrow)

  return nil
}
```

When tokens need to be escrowed in `sendTransfer`, then `escrowToken` is called; when tokens need to be unescrowed on execution of the `OnRecvPacket`, `OnAcknowledgementPacket` or `OnTimeoutPacket` callbacks, then `unescrowToken` is called.

### gRPC query endpoint and CLI to retrieve amount

A gRPC query endpoint is added so that it is possible to retrieve the total amount for a given denomination:

```proto
// TotalEscrowForDenom returns the total amount of tokens in escrow based on the denom.
rpc TotalEscrowForDenom(QueryTotalEscrowForDenomRequest) returns (QueryTotalEscrowForDenomResponse) {
  option (google.api.http).get = "/ibc/apps/transfer/v1/total_escrow/{denom=**}";
}

// QueryTotalEscrowForDenomRequest is the request type for TotalEscrowForDenom RPC method.
message QueryTotalEscrowForDenomRequest {
  string denom = 1;
}

// QueryTotalEscrowForDenomResponse is the response type for TotalEscrowForDenom RPC method.
message QueryTotalEscrowForDenomResponse {
  cosmos.base.v1beta1.Coin amount = 1 [(gogoproto.nullable) = false];
}
```

And a CLI query is also available to retrieve the total amount via the command line:

```shell
query ibc-transfer total-escrow [denom]
```

## Consequences

### Positive

- Possibility to retrieve the total amount of a particular denomination in escrow across all transfer channels without iteration.

### Negative

No notable consequences

### Neutral

- A new entry is added to state for every denomination that is transferred out of the chain.

## References

Issues:

- [#2664](https://github.com/cosmos/ibc-go/issues/2664)

PRs:

- [#3019](https://github.com/cosmos/ibc-go/pull/3019)
- [#3558](https://github.com/cosmos/ibc-go/pull/3558)
