---
title: Events
sidebar_label: Events
sidebar_position: 4
slug: /middleware/callbacks/events
---

# Events

An overview of all events related to the callbacks middleware. There are two types of events, `"ibc_src_callback"` and `"ibc_dest_callback"`.

## Shared Attributes

Both of these event types share the following attributes:

|     **Attribute Key**     |                                   **Attribute Values**                                  |    **Optional**    |
|:-------------------------:|:---------------------------------------------------------------------------------------:|:------------------:|
|           module          |                                      "ibccallbacks"                                     |                    |
|        callback_type      | **One of**: "send_packet", "acknowledgement_packet", "timeout_packet", "receive_packet" |                    |
|      callback_address     |                                          string                                         |                    |
|  callback_exec_gas_limit  |                               string (parsed from uint64)                               |                    |
| callback_commit_gas_limit |                               string (parsed from uint64)                               |                    |
|      packet_sequence      |                               string (parsed from uint64)                               |                    |
|      callback_result      |                             **One of**: "success", "failure"                            |                    |
|       callback_error      |                            string (parsed from callback err)                            | Yes, if err != nil |

## `ibc_src_callback` Attributes

|  **Attribute Key** |   **Attribute Values**   |
|:------------------:|:------------------------:|
|   packet_src_port  |   string (sourcePortID)  |
| packet_src_channel | string (sourceChannelID) |

## `ibc_dest_callback` Attributes

|  **Attribute Key**  |   **Attribute Values**   |
|:-------------------:|:------------------------:|
|   packet_dest_port  |   string (destPortID)    |
| packet_dest_channel | string (destChannelID)   |
