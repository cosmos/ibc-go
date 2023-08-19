<!--
order: 5
-->

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

It achieves the first goal because if the relayer reserves user defined gas limit for the callback execution, then even if the callback execution runs out of gas, the packet lifecycle will not be blocked and callback will not be executed again.

It achieves the second goal because if the relayer does not reserve user defined gas limit for the callback execution and the callback runs out of gas, then the entire tx will be reverted and the packet lifecycle will be blocked. This will allow the relayer to retry with a higher gas limit.
