---
title: Active Channels
sidebar_label: Active Channels
sidebar_position: 9
slug: /apps/interchain-accounts/active-channels
---

# Understanding Active Channels

The Interchain Accounts module uses either [ORDERED or UNORDERED](https://github.com/cosmos/ibc/tree/master/spec/core/ics-004-channel-and-packet-semantics#ordering) channels. 

When using `ORDERED` channels, the order of transactions when sending packets from a controller to a host chain is maintained.

When using `UNORDERED` channels, there is no guarantee that the order of transactions when sending packets from the controller to the host chain is maintained. If no ordering is specified in `MsgRegisterInterchainAccount`, then the default ordering for new ICA channels is `UNORDERED`.

> A limitation when using ORDERED channels is that when a packet times out the channel will be closed.

In the case of a channel closing, a controller chain needs to be able to regain access to the interchain account registered on this channel. `Active Channels` enable this functionality.

When an Interchain Account is registered using `MsgRegisterInterchainAccount`, a new channel is created on a particular port. During the `OnChanOpenAck` and `OnChanOpenConfirm` steps (on controller & host chain respectively) the `Active Channel` for this interchain account is stored in state.

It is possible to create a new channel using the same controller chain portID if the previously set `Active Channel` is now in a `CLOSED` state. This channel creation can be initialized programmatically by sending a new `MsgChannelOpenInit` message like so:

```go
msg := channeltypes.NewMsgChannelOpenInit(portID, string(versionBytes), channeltypes.ORDERED, []string{connectionID}, icatypes.HostPortID, authtypes.NewModuleAddress(icatypes.ModuleName).String())
handler := keeper.msgRouter.Handler(msg)
res, err := handler(ctx, msg)
if err != nil {
  return err
}
```

Alternatively, any relayer operator may initiate a new channel handshake for this interchain account once the previously set `Active Channel` is in a `CLOSED` state. This is done by initiating the channel handshake on the controller chain using the same portID associated with the interchain account in question.  

It is important to note that once a channel has been opened for a given interchain account, new channels can not be opened for this account until the currently set `Active Channel` is set to `CLOSED`.

## Future improvements

Future versions of the ICS-27 protocol and the Interchain Accounts module will likely use a new channel type that provides ordering of packets without the channel closing in the event of a packet timing out, thus removing the need for `Active Channels` entirely.
The following is a list of issues which will provide the infrastructure to make this possible:

- [IBC Channel Upgrades](https://github.com/cosmos/ibc-go/issues/1599)
- [Implement ORDERED_ALLOW_TIMEOUT logic in 04-channel](https://github.com/cosmos/ibc-go/issues/1661)
- [Add ORDERED_ALLOW_TIMEOUT as supported ordering in 03-connection](https://github.com/cosmos/ibc-go/issues/1662)
- [Allow ICA channels to be opened as ORDERED_ALLOW_TIMEOUT](https://github.com/cosmos/ibc-go/issues/1663)
