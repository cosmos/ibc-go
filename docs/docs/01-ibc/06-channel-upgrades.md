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

It is possible to cancel a channel upgrade in if the following are true:

- the channel has not yet reached the `FLUSHCOMPLETE` state.
- the upgrade has initiated. This will be true if the `MsgChannelUpgradeInit` or `MsgChannelUpgradeTry` message has been
  submitted.
- An `ErrorReceipt` of a failed upgrade attempt on the counterparty chain and proof are provided.

> Note: if the signer is the authority, e.g. the `gov` address, no `ErrorReceipt` or proof is required.
> These can be left empty in the `MsgChannelUpgradeCancel` message.

Upon cancelling a channel upgrade, an `ErrorReceipt` will be written with the current channel's upgrade sequence, and
the channel will be reverted to the pre-upgraded state.

The application callback's `OnChanUpgradeRestore` method will be invoked.

It will then be possible to re-initiate the upgrade by sending a `MsgChannelOpenInit` message.
