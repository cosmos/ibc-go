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

Channel upgradability is an IBC-level protocol that allows chains to leverage new channel features without having to create new channels or perform a network-wide upgrade. 

Prior to this feature, developers who wanted to update an application module or add a middleware to their application flow would need to negotiate a new channel in order to use the updated application feature/middleware, resulting in a loss of the accumulated state/liquidity, token fungibility (as the channel would have been encoded in the IBC denom), and any other larger network effects of losing usage of the existing channel from relayers monitoring, etc.

With channel upgradability, applications will be able to implement features such as but not limited to: potentially adding [denom metadata to tokens](https://github.com/cosmos/ibc/discussions/719), or utilizing the [fee middleware](https://github.com/cosmos/ibc/tree/main/spec/app/ics-029-fee-payment), all while maintaining the channels on which they currently operate.

This document outlines the channel upgrade feature, and the multiple steps used in the upgrade process.

## Channel Upgrade Handshake

Channel upgrades will be initialized using a handshake process that is designed to be similar to the standard connection/channel opening handshake.

```go
type Channel struct {
	// current state of the channel end
	State State `protobuf:"varint,1,opt,name=state,proto3,enum=ibc.core.channel.v1.State" json:"state,omitempty"`
	// whether the channel is ordered or unordered
	Ordering Order `protobuf:"varint,2,opt,name=ordering,proto3,enum=ibc.core.channel.v1.Order" json:"ordering,omitempty"`
	// counterparty channel end
	Counterparty Counterparty `protobuf:"bytes,3,opt,name=counterparty,proto3" json:"counterparty"`
	// list of connection identifiers, in order, along which packets sent on
	// this channel will travel
	ConnectionHops []string `protobuf:"bytes,4,rep,name=connection_hops,json=connectionHops,proto3" json:"connection_hops,omitempty"`
	// opaque channel version, which is agreed upon during the handshake
	Version string `protobuf:"bytes,5,opt,name=version,proto3" json:"version,omitempty"`
	// upgrade sequence indicates the latest upgrade attempt performed by this channel
	// the value of 0 indicates the channel has never been upgraded
	UpgradeSequence uint64 `protobuf:"varint,6,opt,name=upgrade_sequence,json=upgradeSequence,proto3" json:"upgrade_sequence,omitempty"`
}
```

The version, connection hops, and channel ordering are fields in this channel struct which can be changed. For example, the fee middleware can be added to an application module by updating the version string [shown here](https://github.com/cosmos/ibc-go/blob/995b647381b909e9d6065d6c21004f18fab37f55/modules/apps/29-fee/types/metadata.pb.go#L28). However, although connection hops can change in a channel upgrade, both sides must still be each other's counterparty.

On a high level, successful handshake process for channel upgrades works as follows:

1. The chain initiating the upgrade process (chain A) submits the `ChanUpgradeInit` function.
2. Subsequently, the counterparty (chain B) changes its channel end from `OPEN` to `FLUSHING` with `ChanUpgradeTry`.
3. Upon successful completion of the previous step, chain A sets its channel state from `OPEN` to `FLUSHING` or `FLUSHCOMPLETE` with `ChanUpgradeAck` depending on the packet flushing status
4. Finally, chain B sets its channel state to `FLUSHCOMPLETE` when its packet flush is finished, and then `OPEN` with `ChanUpgradeConfirm`.

Each handshake step will be documented below in greater detail.

## Cancelling a Channel Upgrade

Channel upgrade cancellation is performed by submitting a `MsgChannelUpgradeCancel` message.

It is possible for the authority to cancel an in-progress channel upgrade if the following are true:

- The signer is the authority
- The channel state has not reached FLUSHCOMPLETE
- If the channel state has reached FLUSHCOMPLETE, an existence proof of an `ErrorReceipt` on the counterparty chain is provided at our upgrade sequence or greater

It is possible for a relayer to cancel an in-progress channel upgrade if the following are true:
- An existence proof of an `ErrorReceipt` on the counterparty chain is provided at our upgrade sequence or gr   eater

> Note: if the signer is the authority, e.g. the `gov` address, no `ErrorReceipt` or proof is required if the current channel state is not in FLUSHCOMPLETE.
> These can be left empty in the `MsgChannelUpgradeCancel` message in that case.

Upon cancelling a channel upgrade, an `ErrorReceipt` will be written with the channel's current upgrade sequence, and
the channel will move back to `OPEN` state keeping its original parameters.

The application's `OnChanUpgradeRestore` callback method will be invoked.

It will then be possible to re-initiate an upgrade by sending a `MsgChannelOpenInit` message.
