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

Channel upgradeability is an IBC-level protocol that allows chains to leverage new channel features without having to create new channels or perform a network-wide upgrade. Prior to this feature, developers who wanted to update an application module or add a middleware to their application flow would need to negotiate a new channel in order to use the updated application feature/middleware, resulting in a loss of the accumulated state/liquidity, token fungibility (as the channel would have been encoded in the IBC denom), and any other larger network effects of losing usage of the existing channel from relayers monitoring, etc.

With channel upgradeability, applications will be able to implement features such as but not limited to: [including a memo field in the packet data for fungible tokens](https://github.com/cosmos/ibc/pull/842), adding [denom metadata to tokens](https://github.com/cosmos/ibc/discussions/719), or utilizing the [fee middleware](https://github.com/cosmos/ibc/tree/main/spec/app/ics-029-fee-payment), all while maintaining the channels on which they currently operate.

This document outlines the channel upgrade feature, and the multiple steps used in the upgrade process.

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
