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

Channel upgradability is an IBC-level protocol that allows chains to leverage new application and channel features without having to create new channels or perform a network-wide upgrade. 

Prior to this feature, developers who wanted to update an application module or add a middleware to their application flow would need to create a new channel in order to use the updated application feature/middleware, resulting in a loss of the accumulated state/liquidity, token fungibility (as the channel ID is encoded in the IBC denom), and any other larger network effects of losing usage of the existing channel from relayers monitoring, etc.

With channel upgradability, applications will be able to implement features such as but not limited to: potentially adding [denom metadata to tokens](https://github.com/cosmos/ibc/discussions/719), or utilizing the [fee middleware](https://github.com/cosmos/ibc/tree/main/spec/app/ics-029-fee-payment), all while maintaining the channels on which they currently operate.

This document outlines the channel upgrade feature, and the multiple steps used in the upgrade process.

## Channel Upgrade Handshake

Channel upgrades will be achieved using a handshake process that is designed to be similar to the standard connection/channel opening handshake.

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

The version, connection hops, and channel ordering are fields in this channel struct which can be changed. For example, the fee middleware can be added to an application module by updating the version string [shown here](https://github.com/cosmos/ibc-go/blob/995b647381b909e9d6065d6c21004f18fab37f55/modules/apps/29-fee/types/metadata.pb.go#L28). However, although connection hops can change in a channel upgrade, both sides must still be each other's counterparty. This is enforced by the upgrade protocol and upgrade attempts which try to alter an expected counterparty will fail.

On a high level, successful handshake process for channel upgrades works as follows:

1. The chain initiating the upgrade process will propose an upgrade.
2. If the counterparty agrees with the proposal, it will block sends and begin flushing any in-flight packets on its channel end. This flushing process will be covered in more detail below.
3. Upon successful completion of the previous step, the initiating chain will also block packet sends and begin flushing any in-flight packets on its channel end. 
4. Once both channel ends have completed flushing packets within the upgrade timeout window, both channel ends can be opened and upgraded to the new channel fields. 

Each handshake step will be documented below in greater detail.

## Initializing a Channel Upgrade

A channel upgrade is initialised by submitting the `MsgChannelUpgradeInit` message, which can be submitted only by the chain itself upon governance authorization. It is possible to upgrade the channel ordering, the channel connection hops, and the channel version, which can be found in the `UpgradeFields`. 

```go
type MsgChannelUpgradeInit struct {
  PortId    string
  ChannelId string
  Fields    UpgradeFields
  Signer    string
}
```

As part of the handling of the `MsgChannelUpgradeInit` message, the application's callbacks `OnChanUpgradeInit` will be triggered as well.

After this message is handled successfully, the channel's upgrade sequence will be incremented. This upgrade sequence will serve as a nonce for the upgrade process to provide replay protection.

### Governance gating on `ChanUpgradeInit`

The message signer for `MsgChannelUpgradeInit` must be the address which has been designated as the `authority` of the `IBCKeeper`. If this proposal passes, the counterparty's channel will upgrade by default.

If chains want to initiate the upgrade of many channels, they will need to submit a governance proposal with multiple `MsgChannelUpgradeInit`  messages, one for each channel they would like to upgrade, again with message signer as the designated `authority` of the `IBCKeeper`

## Channel State and Packet Flushing

`FLUSHING` and `FLUSHCOMPLETE` are additional channel states which have been added to enable the upgrade feature.

These states may consist of: 

```go
const (
  // Default State
  UNINITIALIZED State = 0
  // A channel has just started the opening handshake.
  INIT State = 1
  // A channel has acknowledged the handshake step on the counterparty chain.
  TRYOPEN State = 2
  // A channel has completed the handshake. Open channels are
  // ready to send and receive packets.
  OPEN State = 3
  // A channel has been closed and can no longer be used to send or receive
  // packets.
  CLOSED State = 4
  // A channel has just accepted the upgrade handshake attempt and is flushing in-flight packets.
  FLUSHING State = 5
  // A channel has just completed flushing any in-flight packets.
  FLUSHCOMPLETE State = 6
)
```

These are found in `State` on the `Channel` struct:

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

This will also set the upgrade timeout for the counterparty (i.e. the timeout before which the counterparty chain must move from `FLUSHING` to `FLUSHCOMPLETE`; if it doesn't then the chain will cancel the upgrade and write an error receipt). The timeout is a relative time duration in nanoseconds that can be changed with `MsgUpdateParams` and by default is 10 minutes.

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

## Timing Out a Channel Upgrade

Timing out an outstanding channel upgrade may be necessary during the flushing packet stage of the channel upgrade process. As stated above, with `ChanUpgradeTry` or `ChanUpgradeAck`, the channel state has been changed from `OPEN` to `FLUSHING`, so no new packets will be allowed to be sent over this channel while flushing. If upgrades cannot be performed in a timely manner (due to unforeseen flushing issues), upgrade timeouts allow the channel to avoid blocking packet sends indefinitely. If flushing exceeds the time limit set in the `UpgradeTimeout` channel `Params`, the upgrade process will need to be timed out to abort the upgrade attempt and resume normal channel processing.

Channel upgrades require setting a valid timeout value in the channel `Params` before submitting a `MsgChannelUpgradeTry` or `MsgChannelUpgradeAck` message (by default, 10 minutes): 

```go
type Params struct {
  UpgradeTimeout Timeout 
}
```

A valid Timeout contains either one or both of a timestamp and block height (sequence).

```go
type Timeout struct {
  // block height after which the packet or upgrade times out
  Height types.Height
  // block timestamp (in nanoseconds) after which the packet or upgrade times out
  Timestamp uint64
}
```

This timeout will then be set as a field on the `Upgrade` struct itself when flushing is started.

```go
type Upgrade struct {
  Fields           UpgradeFields 
  Timeout          Timeout       
  NextSequenceSend uint64        
}
```

If the timeout has been exceeded during flushing, a chain can then submit the `MsgChannelUpgradeTimeout` to timeout the channel upgrade process:

```go
type MsgChannelUpgradeTimeout struct {
  PortId              string    
  ChannelId           string
  CounterpartyChannel Channel 
  ProofChannel        []byte
  ProofHeight         types.Height
  Signer              string 
}
```

An `ErrorReceipt` will be written with the channel's current upgrade sequence, and the channel will move back to `OPEN` state keeping its original parameters.

The application's `OnChanUpgradeRestore` callback method will also be invoked.

Note that timing out a channel upgrade will end the upgrade process, and a new `MsgChannelUpgradeInit` will have to be submitted via governance in order to restart the upgrade process.

## IBC App Recommendations

IBC application callbacks should be primarily used to validate data fields and do compatibility checks.

`OnChanUpgradeInit` should validate the proposed version, order, and connection hops, and should return the application version to upgrade to.

`OnChanUpgradeTry` should validate the proposed version (provided by the counterparty), order, and connection hops. The desired upgrade version should be returned.

`OnChanUpgradeAck` should validate the version proposed by the counterparty.

`OnChanUpgradeOpen` should perform any logic associated with changing of the channel fields.

`OnChanUpgradeRestore`  should perform any logic that needs to be executed when an upgrade attempt fails as is reverted.

> IBC applications should not attempt to process any packet data under the new conditions until after `OnChanUpgradeOpen`
> has been executed, as up until this point it is still possible for the upgrade handshake to fail and for the channel
> to remain in the pre-upgraded state. 
