---
title: End Users
sidebar_label: End Users
sidebar_position: 5
slug: /middleware/callbacks/end-users
---

# Usage

This section explains how to use the callbacks middleware from the perspective of an IBC Actor. Callbacks middleware provides two types of callbacks:

- Source callbacks:
    - `SendPacket` callback
    - `OnAcknowledgementPacket` callback
    - `OnTimeoutPacket` callback
- Destination callbacks:
    - `ReceivePacket` callback

For a given channel, the source callbacks are supported if the source chain has the callbacks middleware wired up in the channel's IBC stack. Similarly, the destination callbacks are supported if the destination chain has the callbacks middleware wired up in the channel's IBC stack.

::: tip
Callbacks are always executed after the packet has been processed by the underlying IBC module.
:::

::: warning
If the underlying application module is doing an asynchronous acknowledgement on packet receive (for example, if the [packet forward middleware](https://github.com/cosmos/ibc-apps/tree/main/middleware/packet-forward-middleware) is in the stack, and is being used by this packet), then the callbacks middleware will execute the `ReceivePacket` callback after the acknowledgement has been received.
:::

## Source Callbacks

Source callbacks are natively supported in the following ibc modules (if they are wrapped by the callbacks middleware):

- `transfer`
- `icacontroller`

To have your source callbacks be processed by the callbacks middleware, you must set the memo in the application's packet data to the following format:

```jsonc
{
  "src_callback": {
    "address": "callbackAddressString",
    // optional
    "gas_limit": "userDefinedGasLimitString",
  }
}
```

## Destination Callbacks

Destination callbacks are natively only supported in the transfer module. Note that wrapping icahost is not supported. This is because icahost should be able to execute an arbitrary transaction anyway, and can call contracts or modules directly.

To have your destination callbacks processed by the callbacks middleware, you must set the memo in the application's packet data to the following format:

```jsonc
{
  "dest_callback": {
    "address": "callbackAddressString",
    // optional
    "gas_limit": "userDefinedGasLimitString",
  }
}
```

Note that a packet can have both a source and destination callback.

```jsonc
{
  "src_callback": {
    "address": "callbackAddressString",
    // optional
    "gas_limit": "userDefinedGasLimitString",
  },
  "dest_callback": {
    "address": "callbackAddressString",
    // optional
    "gas_limit": "userDefinedGasLimitString",
  }
}
```

# User Defined Gas Limit

User defined gas limit was added for the following reasons:

- To prevent callbacks from blocking packet lifecycle.
- To prevent relayers from being able to DOS the callback execution by sending a packet with a low amount of gas.

::: tip
There is a chain wide parameter that sets the maximum gas limit that a user can set for a callback. This is to prevent a user from setting a gas limit that is too high for relayers. If the `"gas_limit"` is not set in the packet memo, then the maximum gas limit is used.
:::

These goals are achieved by creating a minimum gas amount required for callback execution. If the relayer provides at least the minimum gas limit for the callback execution, then the packet lifecycle will not be blocked if the callback runs out of gas during execution, and the callback cannot be retried. If the relayer does not provided the minimum amount of gas and the callback executions runs out of gas, the entire tx is reverted and it may be executed again.

::: tip
`SendPacket` callback is always reverted if the callback execution fails or returns an error for any reason. This is so that the packet is not sent if the callback execution fails.
:::
