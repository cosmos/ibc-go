---
title: Events
sidebar_label: Events
sidebar_position: 5
slug: /apps/transfer/ics20-v1/events
---

:::warning
This document is relevant only for fungible token transfers over channels on v1 of the ICS-20 protocol.
:::

# Events

## `MsgTransfer`

| Type         | Attribute Key   | Attribute Value |
|--------------|-----------------|-----------------|
| ibc_transfer | sender          | \{sender\}      |
| ibc_transfer | receiver        | \{receiver\}    |
| ibc_transfer | tokens          | \{jsonTokens\}  |
| ibc_transfer | memo            | \{memo\}        |
| ibc_transfer | forwarding_hops | `nil`           |
| message      | module          | transfer        |

## `OnRecvPacket` callback

| Type                  | Attribute Key   | Attribute Value |
|-----------------------|-----------------|-----------------|
| fungible_token_packet | sender          | \{sender\}      |
| fungible_token_packet | receiver        | \{receiver\}    |
| fungible_token_packet | tokens          | \{jsonTokens\}  |
| fungible_token_packet | memo            | \{memo\}        |
| fungible_token_packet | forwarding_hops | `nil`           |
| fungible_token_packet | success         | \{ackSuccess\}  |
| fungible_token_packet | error           | \{ackError\}    |
| denom                 | trace_hash      | \{hex_hash\}    |
| denom                 | denom           | \{jsonDenom\}   |
| message               | module          | transfer        |

:::note Deprecated
The `denomination_trace` event has been replaced with `denom` event in ibc-go v9.
:::

## `OnAcknowledgePacket` callback

| Type                  | Attribute Key   | Attribute Value  |
|-----------------------|-----------------|------------------|
| fungible_token_packet | sender          | \{sender\}       |
| fungible_token_packet | receiver        | \{receiver\}     |
| fungible_token_packet | tokens          | \{jsonTokens\}   |
| fungible_token_packet | memo            | \{memo\}         |
| fungible_token_packet | forwarding_hops | `nil`            |
| fungible_token_packet | acknowledgement | \{ack.String()\} |
| fungible_token_packet | success / error | \{ack.Response\} |
| message               | module          | transfer         |

## `OnTimeoutPacket` callback

| Type    | Attribute Key   | Attribute Value |
|---------|-----------------|-----------------|
| timeout | refund_receiver | \{receiver\}    |
| timeout | refund_tokens   | \{jsonTokens\}  |
| timeout | memo            | \{memo\}        |
| timeout | forwarding_hops | `nil`           |
| message | module          | transfer        |
