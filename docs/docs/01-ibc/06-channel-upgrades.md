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

The version, connection hops, and channel ordering are fields in this channel struct which can be changed. For example, the fee middleware can be added to an application module by updating the version string [shown here](https://github.com/cosmos/ibc-go/blob/v8.1.0/modules/apps/29-fee/types/metadata.pb.go#L29). However, although connection hops can change in a channel upgrade, both sides must still be each other's counterparty. This is enforced by the upgrade protocol and upgrade attempts which try to alter an expected counterparty will fail.

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

As part of the handling of the `MsgChannelUpgradeInit` message, the application's `OnChanUpgradeInit` callback will be triggered as well.

After this message is handled successfully, the channel's upgrade sequence will be incremented. This upgrade sequence will serve as a nonce for the upgrade process to provide replay protection.

:::warning
Initiating an upgrade in the same block as opening a channel may potentially prevent the counterparty channel from also opening. 
:::

### Governance gating on `ChanUpgradeInit`

The message signer for `MsgChannelUpgradeInit` must be the address which has been designated as the `authority` of the `IBCKeeper`. If this proposal passes, the counterparty's channel will upgrade by default.

If chains want to initiate the upgrade of many channels, they will need to submit a governance proposal with multiple `MsgChannelUpgradeInit`  messages, one for each channel they would like to upgrade, again with message signer as the designated `authority` of the `IBCKeeper`. The `upgrade-channels` CLI command can be used to submit a proposal that initiates the upgrade of multiple channels; see section [Upgrading channels with the CLI](#upgrading-channels-with-the-cli) below for more information.

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

The state will change to `FLUSHCOMPLETE` once there are no in-flight packets left and the channel end is ready to move to `OPEN`. This flush state will also have an impact on how a channel upgrade can be cancelled, as detailed below.

All other parameters will remain the same during the upgrade handshake until the upgrade handshake completes. When the channel is reset to `OPEN` on a successful upgrade handshake, the relevant fields on the channel end will be switched over to the `UpgradeFields` specified in the upgrade.

:::note

When transitioning a channel from UNORDERED to ORDERED, new packet sends from the channel end which upgrades first will be incapable of being timed out until the counterparty has finished upgrading. 

:::

:::warning

Due to the addition of new channel states, packets can still be received and processed in both `FLUSHING` and `FLUSHCOMPLETE` states.
Packets can also be acknowledged in the `FLUSHING` state. Acknowledging will **not** be possible when the channel is in the `FLUSHCOMPLETE` state, since all packets sent from that channel end have been flushed.
Application developers should consider these new states when implementing application logic that relies on the channel state.
It is still only possible to send packets when the channel is in the `OPEN` state, but sending is disallowed when the channel enters `FLUSHING` and `FLUSHCOMPLETE`. When the channel reopens, sending will be possible again.

:::

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

It will then be possible to re-initiate an upgrade by sending a `MsgChannelOpenInit` message.

:::warning

Performing sequentially an upgrade cancellation, upgrade initialization, and another upgrade cancellation in a single block while the counterparty is in `FLUSHCOMPLETE` will lead to corrupted state.
The counterparty will be unable to cancel its upgrade attempt and will require a manual migration. 
When the counterparty is in `FLUSHCOMPLETE`, it requires a proof that the counterparty cancelled its current upgrade attempt. 
When this cancellation is succeeded by an initialization and cancellation in the same block, it results in the proof of cancellation existing only for the next upgrade attempt, not the current. 

:::

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

Note that timing out a channel upgrade will end the upgrade process, and a new `MsgChannelUpgradeInit` will have to be submitted via governance in order to restart the upgrade process.

## Pruning Acknowledgements

Acknowledgements can be pruned by broadcasting the `MsgPruneAcknowledgements` message.

> Note: It is only possible to prune acknowledgements after a channel has been upgraded, so pruning will fail
> if the channel has not yet been upgraded.

```protobuf
// MsgPruneAcknowledgements defines the request type for the PruneAcknowledgements rpc.
message MsgPruneAcknowledgements {
  option (cosmos.msg.v1.signer)      = "signer";
  option (gogoproto.goproto_getters) = false;

  string port_id    = 1;
  string channel_id = 2;
  uint64 limit      = 3;
  string signer     = 4;
}
```

The `port_id` and `channel_id` specify the port and channel to act on, and the `limit` specifies the upper bound for the number
of acknowledgements and packet receipts to prune.

### CLI Usage

Acknowledgements can be pruned via the cli with the `prune-acknowledgements` command.

```bash
simd tx ibc channel prune-acknowledgements [port] [channel] [limit]
```

## IBC App Recommendations

IBC application callbacks should be primarily used to validate data fields and do compatibility checks. Application developers
should be aware that callbacks will be invoked before any core ibc state changes are written.

`OnChanUpgradeInit` should validate the proposed version, order, and connection hops, and should return the application version to upgrade to.

`OnChanUpgradeTry` should validate the proposed version (provided by the counterparty), order, and connection hops. The desired upgrade version should be returned.

`OnChanUpgradeAck` should validate the version proposed by the counterparty.

`OnChanUpgradeOpen` should perform any logic associated with changing of the channel fields.

> IBC applications should not attempt to process any packet data under the new conditions until after `OnChanUpgradeOpen`
> has been executed, as up until this point it is still possible for the upgrade handshake to fail and for the channel
> to remain in the pre-upgraded state. 

## Upgrade an existing transfer application stack to use 29-fee middleware

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

### Submit a governance proposal to execute a MsgChannelUpgradeInit message

> This process can be performed with the new CLI that has been added
> outlined [here](#upgrading-channels-with-the-cli).

Only the configured authority for the ibc module is able to initiate a channel upgrade by submitting a `MsgChannelUpgradeInit` message.

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

> Note: ensure the correct fields.version is specified. This is the new version that the channels will be upgraded to.

### Submit the proposal

```shell
simd tx submit-proposal proposal.json --from <key_or_address>
```

## Upgrading channels with the CLI

A new cli has been added which enables either
    - submitting a governance proposal which contains a `MsgChannelUpgradeInit` for every channel to be upgraded.
    - generating a `proposal.json` file which contains the proposal contents to be edited/submitted at a later date.

The following example, would submit a governance proposal with the specified deposit, title and summary which would
contain a `MsgChannelUpgradeInit` for all `OPEN` channels whose port matches the regular expression `transfer`.

> Note: by adding the `--json` flag, the command would instead output the contents of the proposal which could be 
> stored in a `proposal.json` file to be edited and submitted at a later date.

```bash
simd tx ibc channel upgrade-channels "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}" \
  --deposit "10stake" \
  --title "Channel Upgrades Governance Proposal" \
  --summary "Upgrade all transfer channels to be fee enabled" \
  --port-pattern "transfer"
```

It is also possible to explicitly list a comma separated string of channel IDs. It is important to note that the 
regular expression matching specified by `--port-pattern` (which defaults to `transfer`) still applies.

For example the following command would generate the contents of a `proposal.json` file which would attempt to upgrade
channels with a port ID of `transfer` and a channelID of `channel-0`, `channel-1` or `channel-2`.

```bash
simd tx ibc channel upgrade-channels "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}" \
  --deposit "10stake" \
  --title "Channel Upgrades Governance Proposal" \
  --summary "Upgrade all transfer channels to be fee enabled" \
  --port-pattern "transfer" \
  --channel-ids "channel-0,channel-1,channel-2" \
  --json
```
