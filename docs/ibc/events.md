<!--
order: 5
-->

# Events

**NOTE**: This document is unmaintained and may be out of date!

The IBC module emits the following events. It can be expected that the type `message`,
with an attirbute key of `action` will represent the first event for each message
being processed as emitted by the SDK's baseapp. Each IBC TAO message will
also emit its module name in the format 'ibc_sub-modulename'.

All the events for the Channel handshakes, `SendPacket`, `RecvPacket`, `AcknowledgePacket`, 
`TimeoutPacket` and `TimeoutOnClose` will emit additional events not specified here due to
callbacks to IBC applications.

## ICS 02 - Client

### MsgCreateClient

| Type          | Attribute Key    | Attribute Value   |
|---------------|------------------|-------------------|
| create_client | client_id        | {clientId}        |
| create_client | client_type      | {clientType}      |
| create_client | consensus_height | {consensusHeight} |
| message       | action           | create_client     |
| message       | module           | ibc_client        |

### MsgUpdateClient

| Type          | Attribute Key    | Attribute Value   |
|---------------|------------------|-------------------|
| update_client | client_id        | {clientId}        |
| update_client | client_type      | {clientType}      |
| update_client | consensus_height | {consensusHeight} |
| update_client | header           | {header}          |
| message       | action           | update_client     |
| message       | module           | ibc_client        |

### MsgSubmitMisbehaviour

| Type                | Attribute Key    | Attribute Value     |
|---------------------|------------------|---------------------|
| client_misbehaviour | client_id        | {clientId}          |
| client_misbehaviour | client_type      | {clientType}        |
| client_misbehaviour | consensus_height | {consensusHeight}   |
| message             | action           | client_misbehaviour |
| message             | module           | evidence            |
| message             | sender           | {senderAddress}     |
| submit_evidence     | evidence_hash    | {evidenceHash}      |

### UpdateClientProposal

| Type                   | Attribute Key    | Attribute Value   |
|------------------------|------------------|-------------------|
| update_client_proposal | client_id        | {clientId}        |
| update_client_proposal | client_type      | {clientType}      |
| update_client_proposal | consensus_height | {consensusHeight} |

### UpgradeProposal

| Type                    | Attribute Key   | Attribute Value   |
|-------------------------|-----------------|-------------------|
| upgrade_client_proposal | title           | {title}           |
| upgrade_client_proposal | height          | {height}          |     

## ICS 03 - Connection

### MsgConnectionOpenInit

| Type                 | Attribute Key              | Attribute Value             |
|----------------------|----------------------------|-----------------------------|
| connection_open_init | connection_id              | {connectionId}              |
| connection_open_init | client_id                  | {clientId}                  |
| connection_open_init | counterparty_client_id     | {counterparty.clientId}     |
| message              | action                     | connection_open_init        |
| message              | module                     | ibc_connection              |

### MsgConnectionOpenTry

| Type                | Attribute Key              | Attribute Value             |
|---------------------|----------------------------|-----------------------------|
| connection_open_try | connection_id              | {connectionId}              | 
| connection_open_try | client_id                  | {clientId}                  |
| connection_open_try | counterparty_client_id     | {counterparty.clientId      |
| connection_open_try | counterparty_connection_id | {counterparty.connectionId} |
| message             | action                     | connection_open_try         |
| message             | module                     | ibc_connection              |

### MsgConnectionOpenAck

| Type                 | Attribute Key              | Attribute Value             |
|----------------------|----------------------------|-----------------------------|
| connection_open_ack  | connection_id              | {connectionId}              |
| connection_open_ack  | client_id                  | {clientId}                  |
| connection_open_ack  | counterparty_client_id     | {counterparty.clientId}     |
| connection_open_ack  | counterparty_connection_id | {counterparty.connectionId} |
| message              | module                     | ibc_connection              |
| message              | action                     | connection_open_ack         |

### MsgConnectionOpenConfirm

| Type                    | Attribute Key              | Attribute Value             |
|-------------------------|----------------------------|-----------------------------|
| connection_open_confirm | connection_id              | {connectionId}              |
| connection_open_confirm | client_id                  | {clientId}                  |
| connection_open_confirm | counterparty_client_id     | {counterparty.clientId}     |
| connection_open_confirm | counterparty_connection_id | {counterparty.connectionId} |
| message                 | action                     | connection_open_confirm     |
| message                 | module                     | ibc_connection              |

## ICS 04 - Channel

### MsgChannelOpenInit

| Type              | Attribute Key           | Attribute Value                  |
|-------------------|-------------------------|----------------------------------|
| channel_open_init | port_id                 | {portId}                         |
| channel_open_init | channel_id              | {channelId}                      |
| channel_open_init | counterparty_port_id    | {channel.counterparty.portId}    |
| channel_open_init | connection_id           | {channel.connectionHops}         |
| message           | action                  | channel_open_init                |
| message           | module                  | ibc_channel                      |

### MsgChannelOpenTry

| Type             | Attribute Key           | Attribute Value                  |
|------------------|-------------------------|----------------------------------|
| channel_open_try | port_id                 | {portId}                         |
| channel_open_try | channel_id              | {channelId}                      |
| channel_open_try | counterparty_port_id    | {channel.counterparty.portId}    |
| channel_open_try | counterparty_channel_id | {channel.counterparty.channelId} |
| channel_open_try | connection_id           | {channel.connectionHops}         |
| message          | action                  | channel_open_try                 |
| message          | module                  | ibc_channel                      |

### MsgChannelOpenAck

| Type             | Attribute Key           | Attribute Value                  |
|------------------|-------------------------|----------------------------------|
| channel_open_ack | port_id                 | {portId}                         |
| channel_open_ack | channel_id              | {channelId}                      |
| channel_open_ack | counterparty_port_id    | {channel.counterparty.portId}    |
| channel_open_ack | counterparty_channel_id | {channel.counterparty.channelId} |
| channel_open_ack | connection_id           | {channel.connectionHops}         |
| message          | action                  | channel_open_ack                 |
| message          | module                  | ibc_channel                      |

### MsgChannelOpenConfirm

| Type                 | Attribute Key           | Attribute Value                  |
|----------------------|-------------------------|----------------------------------|
| channel_open_confirm | port_id                 | {portId}                         |
| channel_open_confirm | channel_id              | {channelId}                      |
| channel_open_confirm | counterparty_port_id    | {channel.counterparty.portId}    |
| channel_open_confirm | counterparty_channel_id | {channel.counterparty.channelId} |
| channel_open_confirm | connection_id           | {channel.connectionHops}         |
| message              | module                  | ibc_channel                      |
| message              | action                  | channel_open_confirm             |

### MsgChannelCloseInit

| Type               | Attribute Key           | Attribute Value                  |
|--------------------|-------------------------|----------------------------------|
| channel_close_init | port_id                 | {portId}                         |
| channel_close_init | channel_id              | {channelId}                      |
| channel_close_init | counterparty_port_id    | {channel.counterparty.portId}    |
| channel_close_init | counterparty_channel_id | {channel.counterparty.channelId} |
| channel_close_init | connection_id           | {channel.connectionHops}         |
| message            | action                  | channel_close_init               |
| message            | module                  | ibc_channel                      |

### MsgChannelCloseConfirm

| Type                  | Attribute Key           | Attribute Value                  |
|-----------------------|-------------------------|----------------------------------|
| channel_close_confirm | port_id                 | {portId}                         |
| channel_close_confirm | channel_id              | {channelId}                      |
| channel_close_confirm | counterparty_port_id    | {channel.counterparty.portId}    |
| channel_close_confirm | counterparty_channel_id | {channel.counterparty.channelId} |
| channel_close_confirm | connection_id           | {channel.connectionHops}         |
| message               | action                  | channel_close_confirm            |
| message               | module                  | ibc_channel                      |

### SendPacket (application module call)

| Type        | Attribute Key            | Attribute Value                  | Status     |
|-------------|--------------------------|----------------------------------|------------|
| send_packet | packet_data              | {data}                           | Deprecated |
| send_packet | packet_data_hex          | {hex.Encode(data)}               |            |
| send_packet | packet_timeout_height    | {timeoutHeight}                  |            |
| send_packet | packet_timeout_timestamp | {timeoutTimestamp}               |            |
| send_packet | packet_sequence          | {sequence}                       |            |
| send_packet | packet_src_port          | {sourcePort}                     |            |
| send_packet | packet_src_channel       | {sourceChannel}                  |            |
| send_packet | packet_dst_port          | {destinationPort}                |            |
| send_packet | packet_dst_channel       | {destinationChannel}             |            |
| send_packet | packet_channel_ordering  | {channel.Ordering}               |            |
| send_packet | packet_connection        | {channel.ConnectionHops[0]}      | Deprecated |
| send_packet | connection_id            | {channel.ConnectionHops[0]}      |            |
| message     | action                   | application-module-defined-field |            |
| message     | module                   | ibc-channel                      |            |

### MsgRecvPacket 

| Type        | Attribute Key            | Attribute Value               | Status     |
|-------------|--------------------------|-------------------------------|------------|
| recv_packet | packet_data              | {data}                        | Deprecated |
| recv_packet | packet_data_hex          | {hex.Encode(data)}            |            |
| recv_packet | packet_timeout_height    | {timeoutHeight}               |            |
| recv_packet | packet_timeout_timestamp | {timeoutTimestamp}            |            |
| recv_packet | packet_sequence          | {sequence}                    |            |
| recv_packet | packet_src_port          | {sourcePort}                  |            |
| recv_packet | packet_src_channel       | {sourceChannel}               |            |
| recv_packet | packet_dst_port          | {destinationPort}             |            |
| recv_packet | packet_dst_channel       | {destinationChannel}          |            |
| recv_packet | packet_channel_ordering  | {channel.Ordering}            |            |
| recv_packet | packet_connection        | {channel.ConnectionHops[0]}   | Deprecated |
| recv_packet | connection_id            | {channel.ConnectionHops[0]}   |            |
| message     | action                   | recv_packet                   |            |
| message     | module                   | ibc-channel                   |            |

| Type                  | Attribute Key            | Attribute Value               | Status     |
|-----------------------|--------------------------|-------------------------------|------------|
| write_acknowledgement | packet_data              | {data}                        | Deprecated |
| write_acknowledgement | packet_data_hex          | {hex.Encode(data)}            |            |
| write_acknowledgement | packet_timeout_height    | {timeoutHeight}               |            |
| write_acknowledgement | packet_timeout_timestamp | {timeoutTimestamp}            |            |
| write_acknowledgement | packet_sequence          | {sequence}                    |            |
| write_acknowledgement | packet_src_port          | {sourcePort}                  |            |
| write_acknowledgement | packet_src_channel       | {sourceChannel}               |            |
| write_acknowledgement | packet_dst_port          | {destinationPort}             |            |
| write_acknowledgement | packet_dst_channel       | {destinationChannel}          |            |
| write_acknowledgement | packet_ack               | {ack}                         | Deprecated |
| write_acknowledgement | packet_ack_hex           | {hex.Encode(ack)}             |            |
| write_acknowledgement | packet_channel_ordering  | {channel.Ordering}            |            |
| write_acknowledgement | packet_connection        | {channel.ConnectionHops[0]}   | Deprecated |
| write_acknowledgement | connection_id            | {channel.ConnectionHops[0]}   |            |
| message               | action                   | write_acknowledgement         |            |
| message               | module                   | ibc-channel                   |            |

### MsgAcknowledgePacket 

| Type               | Attribute Key            | Attribute Value               | Status     |
|--------------------|--------------------------|-------------------------------|------------|
| acknowledge_packet | packet_timeout_height    | {timeoutHeight}               |            |
| acknowledge_packet | packet_timeout_timestamp | {timeoutTimestamp}            |            | 
| acknowledge_packet | packet_sequence          | {sequence}                    |            |
| acknowledge_packet | packet_src_port          | {sourcePort}                  |            |
| acknowledge_packet | packet_src_channel       | {sourceChannel}               |            |
| acknowledge_packet | packet_dst_port          | {destinationPort}             |            |
| acknowledge_packet | packet_dst_channel       | {destinationChannel}          |            |
| acknowledge_packet | packet_channel_ordering  | {channel.Ordering}            |            |
| acknowledge_packet | packet_connection        | {channel.ConnectionHops[0]}   | Deprecated |
| acknowledge_packet | connection_id            | {channel.ConnectionHops[0]}   |            |
| message            | action                   | acknowledge_packet            |            |
| message            | module                   | ibc-channel                   |            |

### MsgTimeoutPacket & MsgTimeoutOnClose 

| Type           | Attribute Key            | Attribute Value               |
|----------------|--------------------------|-------------------------------|
| timeout_packet | packet_timeout_height    | {timeoutHeight}               |
| timeout_packet | packet_timeout_timestamp | {timeoutTimestamp}            |
| timeout_packet | packet_sequence          | {sequence}                    |
| timeout_packet | packet_src_port          | {sourcePort}                  |
| timeout_packet | packet_src_channel       | {sourceChannel}               |
| timeout_packet | packet_dst_port          | {destinationPort}             |
| timeout_packet | packet_dst_channel       | {destinationChannel}          |
| timeout_packet | packet_channel_ordering  | {channel.Ordering}            |
| timeout_packet | connection_id            | {channel.ConnectionHops[0]}   |
| message        | action                   | timeout_packet                |
| message        | module                   | ibc-channel                   |
