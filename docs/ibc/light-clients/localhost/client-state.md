## ClientState

The localhost `ClientState` maintains a single field used to track the latest sequence of the state machine i.e. the height of the blockchain.

```go
type ClientState struct {
    // the latest height of the blockchain
    LatestHeight clienttypes.Height
}
```

The latest height is updated periodically through the ABCI [`BeginBlock`](https://docs.cosmos.network/v0.47/building-modules/beginblock-endblock) interface, retrieving the current height from the application context.
