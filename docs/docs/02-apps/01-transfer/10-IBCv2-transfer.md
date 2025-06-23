---
title: IBC v2 Transfer
sidebar_label: IBC v2 Transfer
sidebar_position: 10
slug: /apps/transfer/ics20-v1/ibcv2transfer
---

# IBC v2 Transfer

Much of the core business logic of sending and recieving tokens between chains is unchanged between IBC Classic and IBC v2. Some of the key differences to pay attention to are detailed below. 

## No Channel Handshakes, New Packet Format and Encoding Support

- IBC v2 does not establish connection between applications with a channel handshake. Channel identifiers represent Client IDs and are included in the `Payload`
    - The source and destination port must be `"transfer"`
    - The channel IDs [must be valid client IDs](https://github.com/cosmos/ibc-go/blob/main/modules/apps/transfer/v2/ibc_module.go#L46-L47) of the format `{clientID}-{sequence}`, e.g. 08-wasm-007
- The [`Payload`](https://github.com/cosmos/ibc-go/blob/main/modules/core/04-channel/v2/types/packet.pb.go#L146-L158) contains the [`FungibleTokenPacketData`](https://github.com/cosmos/ibc-go/blob/main/modules/apps/transfer/types/packet.pb.go#L28-L39) for a token transfer. 

The code snippet shows the `Payload` struct.

```go
// Payload contains the source and destination ports and payload for the application (version, encoding, raw bytes)
type Payload struct {
	// specifies the source port of the packet, e.g. transfer
	SourcePort string `protobuf:"bytes,1,opt,name=source_port,json=sourcePort,proto3" json:"source_port,omitempty"`
	// specifies the destination port of the packet, e.g. trasnfer
	DestinationPort string `protobuf:"bytes,2,opt,name=destination_port,json=destinationPort,proto3" json:"destination_port,omitempty"`
	// version of the specified application
	Version string `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
	// the encoding used for the provided value, for transfer this could be JSON, protobuf or ABI
	Encoding string `protobuf:"bytes,4,opt,name=encoding,proto3" json:"encoding,omitempty"`
	// the raw bytes for the payload.
	Value []byte `protobuf:"bytes,5,opt,name=value,proto3" json:"value,omitempty"`
}
```

The code snippet shows the structure of the `Payload` bytes for token transfer

```go
// FungibleTokenPacketData defines a struct for the packet payload
// See FungibleTokenPacketData spec:
// https://github.com/cosmos/ibc/tree/master/spec/app/ics-020-fungible-token-transfer#data-structures
type FungibleTokenPacketData struct {
	// the token denomination to be transferred
	Denom string `protobuf:"bytes,1,opt,name=denom,proto3" json:"denom,omitempty"`
	// the token amount to be transferred
	Amount string `protobuf:"bytes,2,opt,name=amount,proto3" json:"amount,omitempty"`
	// the sender address
	Sender string `protobuf:"bytes,3,opt,name=sender,proto3" json:"sender,omitempty"`
	// the recipient address on the destination chain
	Receiver string `protobuf:"bytes,4,opt,name=receiver,proto3" json:"receiver,omitempty"`
	// optional memo
	Memo string `protobuf:"bytes,5,opt,name=memo,proto3" json:"memo,omitempty"`
}
```

## Base Denoms cannot contain slashes

With the new [`Denom`](https://github.com/cosmos/ibc-go/blob/main/modules/apps/transfer/types/token.pb.go#L81-L87) struct, the base denom, i.e. uatom, is seperated from the trace - the path the token has travelled. The trace is presented as an array of [`Hop`](https://github.com/cosmos/ibc-go/blob/main/modules/apps/transfer/types/token.pb.go#L136-L140)s. 

Because IBC v2 no longer uses channels, it is no longer possible to rely on a fixed format for an identifier so using a base denom that contains a "/" is dissallowed. 

## Changes to the application module interface

Instead of implementing token transfer for `port.IBCModule`, IBC v2 uses the new application interface `api.IBCModule`. More information on the interface differences can be found in the [application section](../../01-ibc/03-apps/00-ibcv2apps.md). 

## MsgTransfer Entrypoint

The `MsgTransfer` entrypoint has been retained in order to retain support for the common entrypoint integrated in most existing frontends.

If `MsgTransfer` is used with a clientID as the `msg.SourceChannel` then the handler will automatically use the IBC v2 protocol. It will internally call the `MsgSendPacket` endpoint so that the execution flow is the same in the state machine for all IBC v2 packets while still presenting the same endpoint for users.

Of course, we want to still retain support for sending v2 packets on existing channels. The denominations of tokens once they leave the origin chain are prefixed by the port and channel ID in IBC v1. Moreover, the transfer escrow accounts holding the original tokens are generated from the channel IDs. Thus, if we wish to interact these remote tokens using IBC v2, we must still use the v1 channel identifiers that they were originally sent with.

Thus, `MsgTransfer` has an additional `UseAliasing` boolean field to indicate that we wish to use IBC v2 protocol while still using the old v1 channel identifiers. This enables users to interact with the same tokens, DEX pools, and cross-chain DEFI protocols using the same denominations that they had previously with the IBC v2 protocol. To use the `MsgTransfer` with aliasing we can submit the message like so:

```go
MsgTransfer{
	SourcePort:       "transfer",
	SourceChannel:    "channel-4", //note: we are using an existing v1 channel identiifer
	Token:            "uatom",
	Sender:           {senderAddr},
	Receiver:         {receiverAddr},
	TimeoutHeight:    ZeroHeight, // note: IBC v2 does not use timeout height
	TimeoutTimestamp: 100_000_000,
	Memo:             "",
	UseAliasing:      true, // set aliasing to true so the handler uses IBC v2 instead of IBC v1
}
```
