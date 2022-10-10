<!--
order: 5
-->

# Events

## MsgTransfer

| Type         | Attribute Key | Attribute Value |
|--------------|---------------|-----------------|
| ibc_transfer | sender        | {sender}        |
| ibc_transfer | receiver      | {receiver}      |
| message      | action        | transfer        |
| message      | module        | transfer        |

## OnRecvPacket callback

| Type                  | Attribute Key | Attribute Value |
|-----------------------|---------------|-----------------|
| fungible_token_packet | module        | transfer        |
| fungible_token_packet | receiver      | {receiver}      |
| fungible_token_packet | denom         | {denom}         |
| fungible_token_packet | amount        | {amount}        |
| fungible_token_packet | success       | {ackSuccess}    |
| fungible_token_packet | metadata      | {metadata}      |
| denomination_trace    | trace_hash    | {hex_hash}      |

## OnAcknowledgePacket callback

| Type                  | Attribute Key   | Attribute Value   |
|-----------------------|-----------------|-------------------|
| fungible_token_packet | module          | transfer          |
| fungible_token_packet | receiver        | {receiver}        |
| fungible_token_packet | denom           | {denom}           |
| fungible_token_packet | amount          | {amount}          |
<<<<<<< HEAD:modules/apps/transfer/spec/05_events.md
=======
| fungible_token_packet | metadata        | {metadata}        |
| fungible_token_packet | acknowledgement | {ack.String()}    |
>>>>>>> 82397d6 (Added optional packet metadata to the packet and message types (#2305)):docs/apps/transfer/events.md
| fungible_token_packet | success | error | {ack.Response}    |

## OnTimeoutPacket callback

| Type                  | Attribute Key   | Attribute Value |
|-----------------------|-----------------|-----------------|
| fungible_token_packet | module          | transfer        |
| fungible_token_packet | refund_receiver | {receiver}      |
| fungible_token_packet | denom           | {denom}         |
| fungible_token_packet | amount          | {amount}        |
| fungible_token_packet | metadata        | {metadata}      |
