# ADR 008: Callback to IBC-App Callers

## Changelog
* 2022-08-10: Initial Draft

## Status

Proposed

## Context

IBC was designed with callbacks between core IBC and IBC applications. IBC apps would send a packet to core IBC. When the result of the packet lifecycle eventually resolved into either an acknowledgement or a timeout, the core IBC called a callback on the IBC application so that the IBC application could take action on the basis of the result (e.g. unescrow tokens for ICS-20)

We are now seeing the desire for secondary applications to call into IBC apps as part of their state machine logic and then do some actions on the basis of the packet result.

E.g. Send an ICS-20 packet, and if it is successful, then send an ICA-packet to swap tokens on LP and return funds to sender

This requires a second layer of callbacks. The IBC application already gets the result of the packet from core IBC, but currently there is no standardized way to pass this information on to a caller module/smart contract.

## Decision

Create a standardized callback interface that callers can implement. IBC applications (or middleware that wraps IBC applications) can now call this callback to route the result of the packet/channel handshake from core IBC to the IBC application to the original caller.

IBC will expose the following enum type:

```proto
enum PacketResponse {
    UNSPECIFIED = 0;
    SUCCESS = 1;
    FAILURE = 2;
    TIMEOUT = 3;
}
```

IBC callers may implement the following interface:

```go
type IBCCaller interface {
    // OnChannelOpen will be called on the IBCCaller when the channel opens
    // this will happen either on ChanOpenAck or ChanOpenConfirm
    OnChannelOpen(ctx sdk.Context, portID, channelID, version string)

    // OnChannelClose will be called on the IBCCaller if the channel closes
    // this will be called on either ChanCloseInit or ChanCloseConfirm and if the channel handshake fails on our end
    // NOTE: currently the channel does not automatically close if the counterparty fails the handhshake so callers must be prepared for an OpenInit to never return a callback for the time being
    OnChannelClose(ctx sdk.Context, portID, channelID string)

    // OnPacketResponse will be called on the IBC Caller after the IBC Application handles its own OnAcknowledgementPacket or OnTimeoutPacket callback
    OnPacketResponse(
        ctx sdk.Context,
        packet channeltypes.Packet,
        response PacketResponse,
    )
}
```

IBC Apps or middleware can then call the IBCCaller callbacks like so in their own callbacks:

```go
// this is the module-defined SendPacket function. It may differ from application to application
// e.g. For ICS20 this would be the SendTransfer function
func SendPacket(ctx sdk.Context, packet channeltypes.Packet, caller IBCaller) {
    // do custom logic and send packet

    // store a mapping of the packetID to the caller
    k.setCaller(packetID, caller)
}

func OnAcknowledgementPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    acknowledgement []byte,
    relayer string,
) {
    // application-specific onAcknowledgmentPacket logic

    // unmarshal ack bytes into the acknowledgment interface
    var ack exported.Acknowledgement
    unmarshal(acknowledgement, ack)

    // send acknowledgement success value in packet response
    // to original caller
    caller := k.getCaller(packetID)

    if ack.Success() {
        caller.OnPacketResponse(ctx, packet, SUCCESS)
    } else {
        caller.OnPacketResonse(ctx, packet, FAILURE)
    }
}

func OnTimeoutPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    relayer string,
) {
    // application-specific onTimeoutPacket logic

    // send timeout response to original caller
    caller := k.getCaller(packetID)
    caller.OnPacketResponse(Ctx, packet, TIMEOUT)
}