---
title: ClientState
sidebar_label: ClientState
sidebar_position: 3
slug: /ibc/light-clients/localhost/client-state
---


# `ClientState`

The 09-localhost `ClientState` maintains a single field used to track the latest sequence of the state machine i.e. the height of the blockchain.

```go
type ClientState struct {
  // the latest height of the blockchain
  LatestHeight clienttypes.Height
}
```

The 09-localhost `ClientState` is instantiated in the `InitGenesis` handler of the 02-client submodule in core IBC.
It calls `CreateLocalhostClient`, declaring a new `ClientState` and initializing it with its own client prefixed store.

```go
func (k Keeper) CreateLocalhostClient(ctx sdk.Context) error {
  var clientState localhost.ClientState
  return clientState.Initialize(ctx, k.cdc, k.ClientStore(ctx, exported.LocalhostClientID), nil)
}
```

It is possible to disable the localhost client by removing the `09-localhost` entry from the `allowed_clients` list through governance.

## Client updates

The latest height is updated periodically through the ABCI [`BeginBlock`](https://docs.cosmos.network/v0.47/building-modules/beginblock-endblock) interface of the 02-client submodule in core IBC.

[See `BeginBlocker` in abci.go.](https://github.com/cosmos/ibc-go/blob/09-localhost/modules/core/02-client/abci.go#L12)

```go
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
  // ...

  if clientState, found := k.GetClientState(ctx, exported.Localhost); found {
    if k.GetClientStatus(ctx, clientState, exported.Localhost) == exported.Active {
      k.UpdateLocalhostClient(ctx, clientState)
    }
  }
}
```

The above calls into the 09-localhost `UpdateState` method of the `ClientState` .
It retrieves the current block height from the application context and sets the `LatestHeight` of the 09-localhost client.

```go
func (cs ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) []exported.Height {
  height := clienttypes.GetSelfHeight(ctx)
  cs.LatestHeight = height

  clientStore.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(cdc, &cs))

  return []exported.Height{height}
}
```

Note that the 09-localhost `ClientState` is not updated through the 02-client interface leveraged by conventional IBC light clients.
