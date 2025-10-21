---
title: Callbacks with IBC v2
sidebar_label: Callbacks with IBC v2
sidebar_position: 7
slug: /middleware/callbacks/callbacks-with-v2
---

# Use with IBC v2

This page highlights some of the differences between IBC v2 and IBC classic relevant for the callbacks middleware and how to use the module with IBC v2. More details on middleware for IBC v2 can be found in the [middleware section](../../01-ibc/04-middleware/02-developIBCv2.md). 

## Interfaces 

Some of the interface differences are:

- The callbacks middleware for IBC v2 requires the [`Underlying Application`](../01-callbacks/01-overview.md) to implement the new [`CallbacksCompatibleModuleV2`](https://github.com/cosmos/ibc-go/blob/main/modules/apps/callbacks/types/callbacks.go#L53-L58) interface. 
- `channeltypesv2.Payload` is now used instead of `channeltypes.Packet`
- With IBC classic, the `OnRecvPacket` callback returns the `ack`, whereas v2 returns the `recvResult` which is the [status of the packet](https://github.com/cosmos/ibc-go/blob/main/modules/core/04-channel/v2/types/packet.pb.go#L26-L38): unspecified, success, failue or asynchronous
- `api.WriteAcknowledgementWrapper` is used instead of `ICS4Wrapper.WriteAcknowledgement`. It is only needed if the lower level application is going to write an asynchronous acknowledgement.

## Contract Developers

The wasmd contract keeper enables cosmwasm developers to use the callbacks middleware. The [cosmwasm documentation](https://cosmwasm.cosmos.network/ibc/extensions/callbacks) provides information for contract developers. The IBC v2 callbacks implementation uses a `Payload` but reconstructs an IBC classic `Packet` to preserve the cosmwasm contract keeper interface. Additionally contracts must now handle the IBC v2 `ErrorAcknowledgement` sentinel value in the case of a failure.

The callbacks middleware can be used for transfer + action workflows, for example a transfer and swap on recieve. These workflows require knowledge of the ibc denom that has been recieved. To assist with parsing the ics20 packet, [helper functions](https://github.com/cosmos/solidity-ibc-eureka/blob/a8870b023e58622fb7b3f733572c684851f8e5ee/packages/cosmwasm/ibc-callbacks-helpers/src/ics20.rs#L7-L41) can be found in the solidity-ibc-eureka repository. 

## Integration

An example integration of the callbacks middleware in a transfer stack that is using IBC v2 can be found in the [ibc-go integration section](../../01-ibc/02-integration.md)
