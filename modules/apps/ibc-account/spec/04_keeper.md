<!--
order: 2
-->
# Keeper

## Structure

```go

type TxEncoder func(data interface{}) ([]byte, error)

type Keeper struct {
   ...
   txEncoders map[string]types.TxEncoder
   ...
   router types.Router
}
```

The most important part of the IBC account keeper, as shown above, is the **map of txEncoders** and the **router**. Because ICS-027 specification defines that the chain can send arbitrary tx bytes to the counterparty chain, both chains must define the way that they process the caller's requests or make the tx bytes that the callee can process.

The `TxEncoder` serializes the source chain's tx bytes from any data. And the map of `TxEncoder` has the key, such as `chain-id` and `keeper`, which the keeper uses to send packets. Therefore, it is necessary to know which source chain's transaction is being executed.

`SerializeCosmosTx(cdc codec.BinaryCodec, registry codectypes.InterfaceRegistry)` provides a way to serialize the tx bytes from messages if the destination chain is based on the Cosmos-SDK.

The router is used to delegate the process of handling the message to a module. When a packet which requests a set of transaction bytes to be run is passed, the router deserializes the tx bytes and passes the message to the handler. The keeper checks the result of each message, and if any message returns an error, the entire transaction is aborted, and state change rolled back.

`TryRunTx(ctx sdk.Context, sourcePort, sourceChannel, typ string, data interface{}, timeoutHeight clienttypes.Height, timeoutTimestamp uint64)` method is used to request transactions to be run on the destination chain. This method uses the `typ` parameter from the `txEncoders map`'s key to find the right `txEncoder`. If the `txEncoder` exists, the transaction is serialized and a `RUNTX` packet is sent to the destination chain. The `TryRunTx` also returns the virtual txHash which is used in the 'Hook' section shown below. This virtual txHash is not related to the actual on-chain transaction, but only 'virtually' created so transactions requested by the Hook can be identified.

### IBC Packets

```go

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
    string error = 4;
}
```

The example above shows the IBC packets that are used in ICS-027. `Type` indicates what action the packet is performing. When a `REGISTER` packet type is delivered, the counterparty chain will create an account with the address using the hash of {destPort}/{destChannel}/{packet.data}, assuming a duplicate prior account doesn't exist.

If the account is created successfully, it returns an acknowledgement packet to the origin chain with type `REGISTER` and code `0`. If there's an error, it returns the acknowledgement packet with type `REGISTER` and the code of the resulting error.

When a `RUNTX` type packet is delivered, the counterparty chain will deserialize the tx bytes (packet's data field) in a predefined way.

In this implementation of ICS27 for the Cosmos-SDK, it deserializes the tx bytes into slices of messages and gets the handler from the router and executes and checks the result like described above.

If the all messages are successful, it returns the acknowledgment packet to the chain with type `RUNTX` and code `0`. If there's an error, it returns the acknowledgement packet with type `RUNTX` and the code and error of the first failed message.

### Hook

```go

type IBCAccountHooks interface {
    OnAccountCreated(ctx sdk.Context, sourcePort, sourceChannel string, address sdk.AccAddress)
    OnTxSucceeded(ctx sdk.Context, sourcePort, sourceChannel string, txHash []byte, txBytes []byte)
    OnTxFailed(ctx sdk.Context, sourcePort, sourceChannel string, txHash []byte, txBytes []byte)
}
```

The example above shows the hook for helping developer using the IBC account keeper.

The hook lets the developer know whether the IBC account has been successfully created on the counterparty chain.

After sending the packet with an `IBCAccountPacketData` with the type `REGISTER`, if the acknowledgement packet with the type `REGISTER` and code `0` is delivered, `OnAccountCreated` is executed with the counterparty chain's chain-id and address.

After sending the packet with an `IBCAccountPacketData` with the type `RUNTX`, if the acknowledgement packet with the type `RUNTX` and code `0` is delivered, `OnTxSucceeded` is executed with the counterparty chain's chain-id, virtual tx hash and requested data that is not serialized. Virtual tx hash is used only for internal logic to distinguish the requested tx and it is computed by hashing the tx bytes and sequence of packet. Otherwise, `OnTxFailed` will be executed.
