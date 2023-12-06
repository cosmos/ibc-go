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

## Cancelling a Channel Upgrade

Channel upgrade cancellation is performed by submitting a `MsgChannelUpgradeCancel` message.

It is possible to cancel an in-progress channel upgrade if the following are true:

- The channel has not yet reached the `FLUSHCOMPLETE` state.
- The upgrade has been initiated. This will be true if the `MsgChannelUpgradeInit` or `MsgChannelUpgradeTry` message has been
  submitted.
- Existence proof of an `ErrorReceipt` on the counterparty chain at an appropriate upgrade sequence is submitted for a failed upgrade attempt.

> Note: if the signer is the authority, e.g. the `gov` address, no `ErrorReceipt` or proof is required.
> These can be left empty in the `MsgChannelUpgradeCancel` message.

Upon cancelling a channel upgrade, an `ErrorReceipt` will be written with the channel's current upgrade sequence, and
the channel will be reverted to the pre-upgrade state.

The application's `OnChanUpgradeRestore` callback method will be invoked.

It will then be possible to re-initiate an upgrade by sending a `MsgChannelOpenInit` message.
