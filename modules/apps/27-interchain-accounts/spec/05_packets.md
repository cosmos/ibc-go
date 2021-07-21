<!--
order: 3
-->

# Packets

```proto
message IBCTxRaw {
    bytes body_bytes = 1;
}

message IBCTxBody {
    repeated google.protobuf.Any messages = 1;
}

enum Type {
    REGISTER = 0;
    RUNTX = 1;
}

message IBCAccountPacketData {
    Type type = 1;
    bytes data = 2;
}

message IBCAccountPacketAcknowledgement {
    Type type = 1;
    string chainID = 2;
    uint32 code = 3;
    bytes data = 4;
    string error = 5;
}
```

- `IBCAccountPacketAcknowledgement` returns the result of the packet request back to the chain that sent the packet.
- `IBCAccountPacketData` is sent when the counterparty chain registers a IBCAccount or wants to execute a specific tx through the IBC Account.
- `IBCAccountPacketData` type field displays the behavior requested by the packet. If the type is `REGISTER`, this means request to register a new IBCAccount. In this case, the destination chain can set the IBCAccount's address, but typically it is recommended to refer to the data field to create the address in a deterministic way. If the IBCAccount has been successfully registered, an `IBCAccountPacketAcknowledgment` is returned to the requesting chain with the `Code` field set to `0`. If there was an error, an `IBCAccountPacketAcknowledgment` is returned to the requesting chain with the `Code` field including the error message.
