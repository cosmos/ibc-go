---
title: State
sidebar_label: State
sidebar_position: 2
slug: /apps/transfer/ics20-v1/state
---

# State

The IBC transfer application module keeps state of the port to which the module is binded and the denomination trace information.

- `PortKey`: `0x01 -> ProtocolBuffer(string)`
- `DenomTraceKey`: `0x02 | []bytes(traceHash) -> ProtocolBuffer(Denom)`
- `DenomKey` : `0x03 | []bytes(traceHash) -> ProtocolBuffer(Denom)`
