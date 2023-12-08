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

A channel upgrade is initialised by submitting the `ChanUpgradeInit` message, which can be submitted by relayer or the chain itself upon governance authorization. This message should specify an appropriate timeout window for the upgrade, in consideration of the governance method chosen (more in the section below).

As part of the handling of the `ChanUpgradeInit` message, the application's callbacks `OnChanUpgradeInit` will be triggered as well.

After this message is handled successfully, the channel's upgrade sequence will be incremented. This upgrade sequence will serve as a nonce for the upgrade process to avoid error receipt proof collisions.

### Governance Gating on `ChanUpgradeInit`

This can happen in several ways:

1. **A and B pre-approves certain upgrade types** : 

    A technical committee/DAO on A, elected by chain governance, initiates a governance proposal using the groups module stating their chain pre approves all channel upgrade types of the transfer module from v1 to v2. If this proposal passes, and if chain B has also pre-approved upgrades of this type (transfer v1 to v2), the channel upgrade executes in a permissioned and pre-approved manner. In this case, any relayer can submit the `MsgChanUpgradeInit` and `MsgChanUpgradeTry`.

2. **Governance-gated upgrades on A and pre-approved upgrade type on B**: 
    
    Chain A passes a governance proposal to approve an upgrade and executes `ChanUpgradeInit`. Chain B has pre-approved all upgrades of this type (initiated on A) and so any relayer can submit `MsgChanUpgradeTry` on B.

3. **Governance-gated upgrades on A and B**: 
    
    Chains A and B both gate the upgrade by a full chain governance or a DAO/technical committee via the groups module. Chain A calls `ChanUpgradeInit`. And chain B, within the timeout window specified by A, must call `ChanUpgradeTry`.

It is important to note that in the 3rd case where all upgrades on a counterparty are gated by governance, the channel upgrade timeout window specified on the source chain must take this information into account since governance proposals can span from days to weeks depending on the chain.

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
