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

A channel upgrade is initialised by submitting the `ChanUpgradeInit` message, which can be submitted only by the chain itself upon governance authorization. This message should specify an appropriate timeout window for the upgrade. It is possible to upgrade the channel ordering, the channel connection hops, and the channel version. 

As part of the handling of the `ChanUpgradeInit` message, the application's callbacks `OnChanUpgradeInit` will be triggered as well.

After this message is handled successfully, the channel's upgrade sequence will be incremented. This upgrade sequence will serve as a nonce for the upgrade process to provide replay protection.

### Governance gating on `ChanUpgradeInit`

The message signer for `MsgChanUpgradeInit` must be the address which has been designated as the `authority` of the `IBCKeeper`. If this proposal passes, the counterparty's channel will upgrade by default.

If chains want to initiate the upgrade of many channels, they will need to submit a governance proposal with multiple `MsgChanUpgradeInit`  messages, one for each channel they would like to upgrade, again with message signer as the designated `authority` of the `IBCKeeper`

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

## Upgrade existing transfer stack to be fee enabled

### Wire up the transfer stack and middleware in app.go

In app.go, the existing transfer stack must be wrapped with the fee middleware.

```golang

import (
    // ... 
    ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee"
    ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
    transfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
    porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
    // ...
)

type App struct {
	// ...
	TransferKeeper        ibctransferkeeper.Keeper
	IBCFeeKeeper          ibcfeekeeper.Keeper
	// ..
}

// ...

app.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
    appCodec, keys[ibcfeetypes.StoreKey],
    app.IBCKeeper.ChannelKeeper, // may be replaced with IBC middleware
    app.IBCKeeper.ChannelKeeper,
    app.IBCKeeper.PortKeeper, app.AccountKeeper, app.BankKeeper,
)

// Create Transfer Keeper and pass IBCFeeKeeper as expected Channel and PortKeeper
// since fee middleware will wrap the IBCKeeper for underlying application.
app.TransferKeeper = ibctransferkeeper.NewKeeper(
    appCodec, keys[ibctransfertypes.StoreKey], app.GetSubspace(ibctransfertypes.ModuleName),
    app.IBCFeeKeeper, // ISC4 Wrapper: fee IBC middleware
    app.IBCKeeper.ChannelKeeper, app.IBCKeeper.PortKeeper,
    app.AccountKeeper, app.BankKeeper, scopedTransferKeeper,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)


ibcRouter := porttypes.NewRouter()

// create IBC module from bottom to top of stack
var transferStack porttypes.IBCModule
transferStack = transfer.NewIBCModule(app.TransferKeeper)
transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)

// Add transfer stack to IBC Router
ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
```

### Submit a governance proposal to execute a MsgChanUpgradeInit message

Only an authority is able to initiate a channel upgrade by submitting a `MsgChanUpgradeInit` message.

Execute a governance proposal specifying the relevant fields to perform a channel upgrade.

Update the following json sample, and copy the contents into `proposal.json`.

```json
{
  "title": "Channel upgrade init",
  "summary": "Channel upgrade init",
  "messages": [
    {
      "@type": "/ibc.core.channel.v1.MsgChannelUpgradeInit",
      "signer": "<gov-address>",
      "port_id": "transfer",
      "channel_id": "channel-...",
      "fields": {
        "ordering": "ORDER_UNORDERED",
        "connection_hops": ["connection-0"],
        "version": "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}"
      }
    }
  ],
  "metadata": "<metadata>",
  "deposit": "10stake"
}
```

### Submit the proposal

```shell
simd tx submit-proposal proposal.json --from <key_or_address>
```
