# ADR 004: Lock fee module upon escrow out of balance

## Changelog

- 03/03/2022: initial draft

## Status

Accepted

## Context

The fee module maintains an escrow account for all fees escrowed to incentivize packet relays.
It also tracks each packet fee escrowed separately from the escrow account. This is because the escrow account only maintains a total balance. It has no reference for which coins belonged to which packet fee.
In the presence of a severe bug, it is possible the escrow balance will become out of sync with the packet fees marked as escrowed.
The ICS29 module should be capable of elegantly handling such a scenario.

## Decision

We will allow for the ICS29 module to become "locked" if the escrow balance is determined to be out of sync with the packet fees marked as escrowed.
A "locked" fee module will not allow for packet escrows to occur nor will it distribute fees. All IBC callbacks will skip performing fee logic, similar to fee disabled channels.

Manual intervention will be needed to unlock the fee module.

### Sending side

Special behavior will have to be accounted for in `OnAcknowledgementPacket`. Since the counterparty will continue to send incentivized acknowledgements for fee enabled channels, the acknowledgement will still need to be unmarshalled into an incentivized acknowledgement before calling the underlying application `OnAcknowledgePacket` callback.

When distributing fees, a cached context should be used. If the escrow account balance would become negative, the current state changes should be discarded and the fee module should be locked using the uncached context. This prevents fees from being partially distributed for a given packetID.

### Receiving side

`OnRecvPacket` should remain unaffected by the fee module becoming locked since escrow accounts only affect the sending side.

## Consequences

### Positive

The fee module can be elegantly disabled in the presence of severe bugs.

### Negative

Extra logic is added to account for edge cases which are only possible in the presence of bugs.

### Neutral

## References

Issues:

- [#821](https://github.com/cosmos/ibc-go/issues/821)
- [#860](https://github.com/cosmos/ibc-go/issues/860)

PR's:

- [#1031](https://github.com/cosmos/ibc-go/pull/1031)
- [#1029](https://github.com/cosmos/ibc-go/pull/1029)
- [#1056](https://github.com/cosmos/ibc-go/pull/1056)
