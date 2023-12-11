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

## Initializing a Channel Upgrade

A channel upgrade is initialised by submitting the `ChanUpgradeInit` message, which can be submitted only by the chain itself upon governance authorization. This message should specify an appropriate timeout window for the upgrade.

As part of the handling of the `ChanUpgradeInit` message, the application's callbacks `OnChanUpgradeInit` will be triggered as well.

After this message is handled successfully, the channel's upgrade sequence will be incremented. This upgrade sequence will serve as a nonce for the upgrade process to avoid error receipt proof collisions.

### Governance Gating on `ChanUpgradeInit`

A technical committee/DAO on A, elected by chain governance, initiates a governance proposal using the groups module sending the `MsgChanUpgradeInit`. If this proposal passes, the counterparty channels upgrade executes in a permissioned and pre-approved manner. 

If chains want to initiate the upgrade of many channels, they will need to submit a governance proposal with multiple `MsgChanUpgradeInit`, one for each channel they would like to upgrade.

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
