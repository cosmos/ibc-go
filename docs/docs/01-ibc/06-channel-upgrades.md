---
title: Channel Upgrades
sidebar_label: Channel Upgrades
sidebar_position: 6
slug: /ibc/channel-upgrades
---

# Channel Upgrades

:::note Synopsis
Learn how to upgrade existing IBC channels.
:::

## Channel State and Packet Flushing

`FLUSHING` and `FLUSHCOMPLETE` are additional states which have been added to enable the upgrade feature.

This is found in the channel state, on the `ChannelEnd` interface:

```interface ChannelEnd {
  state: ChannelState
  ordering: ChannelOrder
  counterpartyPortIdentifier: Identifier
  counterpartyChannelIdentifier: Identifier
  connectionHops: [Identifier]
  version: string
  upgradeSequence: uint64
}
```

`startFlushing` is the specific method which is called in `ChanUpgradeTry` and `ChanUpgradeAck` to update the state on the channel end. This will set the timeout on the upgrade and update the channel state to `FLUSHING` which will block the upgrade from continuing until all in-flight packets have been flushed. The state will change to `FLUSHCOMPLETE` once there are no in-flight packets left and the channel end is ready to move to `OPEN`. This flush state will also have an impact on how a channel ugrade can be cancelled, as detailed below.

All other parameters will remain the same during the upgrade handshake until the upgrade handshake completes. When the channel is reset to `OPEN` on a successful upgrade handshake, the relevant fields on the channel end will be switched over to the `UpgradeFields` specified in the upgrade.

## Cancelling a Channel Upgrade

Channel upgrade cancellation is performed by submitting a `MsgChannelUpgradeCancel` message.

It is possible for the authority to cancel an in-progress channel upgrade if the following are true:

- The signer is the authority
- The channel state has not reached FLUSHCOMPLETE
- If the channel state has reached FLUSHCOMPLETE, an existence proof of an `ErrorReceipt` on the counterparty chain is provided at our upgrade sequence or greater

It is possible for a relayer to cancel an in-progress channel upgrade if the following are true:
- An existence proof of an `ErrorReceipt` on the counterparty chain is provided at our upgrade sequence or greater

> Note: if the signer is the authority, e.g. the `gov` address, no `ErrorReceipt` or proof is required if the current channel state is not in FLUSHCOMPLETE.
> These can be left empty in the `MsgChannelUpgradeCancel` message in that case.

Upon cancelling a channel upgrade, an `ErrorReceipt` will be written with the channel's current upgrade sequence, and
the channel will move back to `OPEN` state keeping its original parameters.

The application's `OnChanUpgradeRestore` callback method will be invoked.

It will then be possible to re-initiate an upgrade by sending a `MsgChannelOpenInit` message.
