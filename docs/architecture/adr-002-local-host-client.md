# ADR 002: Local Host Client

### Context

Loopback (or localhost) clients allow two modules on the same chain to communicate over IBC. This is useful if the modules do not have information about where the counterparty module is located. It also provides a uniform interface with which modules can interact with each other, whether they exist on the same chain or not.

The previous localhost implementation in the SDK was broken because it tried to get the connections and channels from the localhost client store, even though they were actually stored in the connection and channel store respectively. The localhost needs read-only access to the full IBC store so that it can efficiently verify the state stored by the local IBC store.

### Decision

- Modify `ClientKeeper.ClientStore` to return the full IBC store with readonly access if `clientID` is localhost.
- Create a ReadOnly wrapper for the KVStore in the SDK

```go
// ClientStore returns isolated prefix store for each client so they can read/write in separate
// namespace without being able to read/write other client's data.
// In the special case of clientID == "localhost", the full store is returned.
func (k Keeper) ClientStore(ctx sdk.Context, clientID string) sdk.KVStore {
    if clientID == "localhost" {
        return readonly(ctx.KVStore(k.storeKey))
    }
	clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
	return prefix.NewStore(ctx.KVStore(k.storeKey), clientPrefix)
}
```

The localhost implementation can remain unchanged, but now it will have access to the entire IBC store, and thus getting connections and channels using the paths specified by `24-host` will work as expected.

All of the special case code in the IBC client handlers can be removed as localhost will not have any separate state.

Since the localhost will have direct access to the current IBC store, the packet commitment does not need to be committed to a block before receiving it. In fact, they can happen in the same block, since the store will contain the packet commitment as soon as the send packet is processed.
Thus, if a relayer knows that a packet is being sent through the localhost client, they can send a `sdk.MultiMsg` transaction that contains the msg that sends the packet, along with `RecvPacket` and `AcknowledgePacket`.

This allows a relayer to execute the entire packet flow in a single tx for localhost clients, rather than waiting for the block to commit for each step in the packet flow. This will make the user experience of sending packets between two modules on the same chain, very much like sending a regular SDK message; although the gas cost may be higher.

### Consequences

### Positive

- A working localhost client
- No special cases in client handlers
- Single transaction packet-flow for local-host clients

### Negative

- Without the readonly change, this would give write access to a potentially faulty client implementation
- The readonly change will require a change in the SDK (though we can temporarily enforce this in localhost itself)