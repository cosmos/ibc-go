---
title: Setting Rate Limits
sidebar_label: Setting Rate Limits
sidebar_position: 3
slug: /apps/rate-limit-middleware/setting-rate-limits
---

# Setting Rate Limits

Rate limits are set through a governance-gated authority on a per denom, and per channel / client basis. To add a rate limit, the [`MsgAddRateLimit`](https://github.com/cosmos/ibc-go/blob/main/modules/apps/rate-limiting/types/msgs.go#L26-L34) message must be executed which includes: 

- Denom: the asset that the rate limit should be applied to
- ChannelOrClientId: the channelID for use with IBC classic connections, or the clientID for use with IBC v2 connections
- MaxPercentSend: the outflow threshold as a percentage of the `channelValue`. More explicitly, a packet being sent would exceed the threshold quota if: (Outflow - Inflow + Packet Amount) / channelValue is greater than MaxPercentSend
- MaxPercentRecv: the inflow threshold as a percentage of the `channelValue`
- DurationHours: the length of time, after which the rate limits reset

## Updating, Removing or Resetting Rate Limits

- If rate limits were set to be too low or high for a given channel/client, they can be updated with [`MsgUpdateRateLimit`](https://github.com/cosmos/ibc-go/blob/main/modules/apps/rate-limiting/types/msgs.go#L81-L89). 
- If rate limits are no longer needed, they can be removed with [`MsgRemoveRateLimit`](https://github.com/cosmos/ibc-go/blob/main/modules/apps/rate-limiting/types/msgs.go#L136-L141).
- If the flow counter needs to be reset for a given rate limit, it is possible to do so with [`MsgResetRateLimit`](https://github.com/cosmos/ibc-go/blob/main/modules/apps/rate-limiting/types/msgs.go#L169-L174).
