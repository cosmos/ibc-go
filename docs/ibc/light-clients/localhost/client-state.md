<!--
order: 2
-->

## ClientState

The 09-localhost `ClientState` maintains a field used to track the latest sequence of the state machine i.e. the height of the blockchain,
and a boolean indicating whether or not the localhost client is enabled.

```go
type ClientState struct {
    // the latest height of the blockchain
    LatestHeight clienttypes.Height
	// whether or not the localhost client is enabled
	Enabled bool
}
```

The 09-localhost `ClientState` is instantiated in the `InitGenesis` handler of the 02-client submodule in core IBC.
It calls `CreateLocalhostClient`, declaring a new `ClientState` and initializing it with its own client prefixed store.
Whether or not it is enabled depends on if `09-localhost` is in the list of `allowed_clients`.

```go
func (k Keeper) CreateLocalhostClient(ctx sdk.Context) error {
    clientState := localhost.NewClientState(types.GetSelfHeight(ctx), k.GetParams(ctx).IsAllowedClient(exported.Localhost))
    return clientState.Initialize(ctx, k.cdc, k.ClientStore(ctx, exported.Localhost), nil)
}
```

It is possible to disable the localhost client by removing the `09-localhost` entry from the `allowed_clients` list through governance
with a `MsgUpdateParams`.

```go
type Params struct {
    // allowed_clients defines the list of allowed client state types.
    AllowedClients []string
}

type MsgUpdateParams struct {
    // authority is the address that controls the module.
    Authority string
    // NOTE: All parameters must be supplied.
    Params Params
}
```

### Updates

The latest height is updated periodically through the ABCI [`BeginBlock`](https://docs.cosmos.network/v0.47/building-modules/beginblock-endblock) interface of the 02-client submodule in core IBC.

[See `BeginBlocker` in abci.go](https://github.com/cosmos/ibc-go/blob/09-localhost/modules/core/02-client/abci.go#L12)

```go
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	// ...

	if clientState, found := k.GetClientState(ctx, exported.Localhost); found {
		k.UpdateLocalhostClient(ctx, clientState)
	}
}
```

The above calls into the the 09-localhost `UpdateState` method of the `ClientState` .
It retrieves the current block height from the application context and sets the `LatestHeight` of the 09-localhost client.

```go
func (cs ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	height := clienttypes.GetSelfHeight(ctx)
	cs.LatestHeight = height

	clientStore.Set([]byte(host.KeyClientState), clienttypes.MustMarshalClientState(cdc, &cs))

	return []exported.Height{height}
}
```

Note that the 09-localhost `ClientState` is not updated through the 02-client interface leveraged by conventional IBC light clients.
