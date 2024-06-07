---
title: Events
sidebar_label: Events
sidebar_position: 5
slug: /apps/transfer/events
---

# Events

## `MsgTransfer`

| Type         | Attribute Key | Attribute Value |
|--------------|---------------|-----------------|
| ibc_transfer | sender        | \{sender\}      |
| ibc_transfer | receiver      | \{receiver\}    |
| ibc_transfer | tokens        | \{jsonTokens\}  |
| ibc_transfer | memo          | \{memo\}        |
| message      | module        | transfer        |

## `OnRecvPacket` callback

| Type                  | Attribute Key | Attribute Value  |
|-----------------------|---------------|------------------|
| fungible_token_packet | sender        | \{sender\}       | 
| fungible_token_packet | receiver      | \{receiver\}     | 
| fungible_token_packet | tokens        | \{jsonTokens\}   |
| fungible_token_packet | success       | \{ackSuccess\}   |
| fungible_token_packet | error         | \{ackError\}     |
| fungible_token_packet | memo          | \{memo\}         | 
| denomination          | trace_hash    | \{hex_hash\}     |
| denomination          | denom         | \{jsonDenom\}    |
| message               | module        | transfer         |

## `OnAcknowledgePacket` callback

| Type                  | Attribute Key   | Attribute Value  |
|-----------------------|-----------------|------------------|
| fungible_token_packet | sender          | \{sender\}       |
| fungible_token_packet | receiver        | \{receiver\}     |
| fungible_token_packet | tokens          | \{jsonTokens\}   |
| fungible_token_packet | memo            | \{memo\}         |
| fungible_token_packet | acknowledgement | \{ack.String()\} |
| fungible_token_packet | success / error | \{ack.Response\} |
| message               | module          | transfer         |

## `OnTimeoutPacket` callback

| Type    | Attribute Key   | Attribute Value |
|---------|-----------------|-----------------|
| timeout | refund_receiver | \{receiver\}    |
| timeout | refund_tokens   | \{jsonTokens\}  |
| timeout | memo            | \{memo\}        |
| message | module          | transfer        |
