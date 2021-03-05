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

### Consequences

### Positive

- A working localhost client

### Negative

- Without the readonly change, this would give write access to a potentially faulty client implementation
- The readonly change will require a change in the SDK