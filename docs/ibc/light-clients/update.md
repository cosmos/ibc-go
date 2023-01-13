<!--
order: 5
-->

# Implementing the `ClientMessage` interface

As mentioned before in the documentation about [implementing the `ConsensusState` interface](./consensus-state.md), [`ClientMessage`](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L145) is an interface used to update an IBC client. This update may be done by a single header, a batch of headers, misbehaviour, or any type which when verified produces a change to the consensus state of the IBC client. This interface has been purposefully kept generic in order to give the maximum amount of flexibility to the light client implementer. 

```golang 
type ClientMessage interface {
	proto.Message

	ClientType() string
	ValidateBasic() error
}
```

The `ClientMessage` will be passed to the client to be used in [`UpdateClient`](https://github.com/cosmos/ibc-go/blob/57da75a70145409247e85365b64a4b2fc6ddad2f/modules/core/02-client/keeper/client.go#L53), which will handle a number of cases including misbehaviour and/or updating the consensus state. However, this `UpdateClient` function will always reference the specific functions determined by the relevant `ClientState`. This is because `UpdateClient` retrieves the client state by client ID (available in `MsgUpdateClient`). This client state implements the `ClientState` interface for a specific client type (e.g. Tendermint). The functions called on the client state instance in `UpdateClient` will be the specific implementations of `VerifyClientMessage`, `CheckForMisbehaviour`, `UpdateStateOnMisbehaviour` and `UpdateState` functions of the `ClientState` interface for that particular client type.