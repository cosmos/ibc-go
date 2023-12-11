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

This is found in the channel state on  `Channel`:

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

`startFlushing` is the specific method which is called in `ChanUpgradeTry` and `ChanUpgradeAck` to update the state on the channel end. This will set the timeout on the upgrade and update the channel state to `FLUSHING` which will block the upgrade from continuing until all in-flight packets have been flushed. 

This will also set the upgrade timeout for the counterparty (i.e. the timeout before which the counterparty chain must move from `FLUSHING` to `FLUSHCOMPLETE`; if it doesn't then the chain will cancel the upgrade and write an error receipt). The timeout is a relative time duration in nanoseconds that can be changed with MsgUpdateParams and by default is 10 minutes.

The state will change to `FLUSHCOMPLETE` once there are no in-flight packets left and the channel end is ready to move to `OPEN`. This flush state will also have an impact on how a channel ugrade can be cancelled, as detailed below.

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
