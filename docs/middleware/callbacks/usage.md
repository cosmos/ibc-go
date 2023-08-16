# Usage

This section explains how to use the callbacks middleware from the perspective of an IBC Actor. Callbacks middleware provides two types of callbacks:

- Source Callbacks:
  - SendPacket Callback
  - OnAcknowledgementPacket Callback
  - OnTimeoutPacket Callback
- Destination Callbacks:
  - ReceivePacket Callback

For a given channel, the source callbacks are supported if the source chain has the callbacks middleware wired up in the channel's ibc stack. Similarly, the destination callbacks are supported if the destination chain has the callbacks middleware wired up in the channel's ibc stack.

::: tip
Callbacks are always executed after the packet has been processed by the underlying IBC module.
:::

::: warning
If the underlying application module is doing an asynchronous acknowledgement on packet receive (for example, if the packet forward middleware is in the stack, and is being used by this packet), then the callbacks middleware will execute the ReceivePacket callback after the acknowledgement has been received.
:::

## Source Callbacks

Source callbacks are natively supported in the following ibc modules (if they are wrapped my the callbacks middleware):

- transfer
- icacontroller

To have your source callbacks be processed by the callbacks middleware, you must set the packet memo to the following format:

```json
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

To have your destination callbacks processed by the callbacks middleware, you must set the packet memo to the following format:

```json
{
  "dest_callback": {
    "address": "callbackAddressString",
    // optional
    "gas_limit": "userDefinedGasLimitString",
  }
}
```

Note that a packet can have both a source and destination callback.

```json
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
