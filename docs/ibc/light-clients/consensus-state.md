<!--
order: 3
-->

# Implementing the `ConsensusState` and `ClientMessage` interfaces

A `ConsensusState` is the snapshot of the counterparty chain that an IBC client uses to verify proofs. The further development of multiple types of IBC light clients and the difficulties presented by this generalization problem (see [ADR-006](https://github.com/cosmos/ibc-go/blob/main/docs/architecture/adr-006-02-client-refactor.md) for more information about this historical context) led to the design decision of each client keeping track of and set its own `ClientState` and `ConsensusState`, as well as the simplification of client `ConsensusState` updates through the generalized `ClientMessage` interface.

The below [`ConsensusState`](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L134) interface is a generalized interface for the types of information a `ConsensusState` could contain. For a reference `ConsensusState` implementation, please see the [Tendermint light client `ConsensusState`](https://github.com/cosmos/ibc-go/blob/main/modules/light-clients/07-tendermint/consensus_state.go).

## `ClientType` method

This is the type of client consensus. It should be the same as the `ClientType` return value for the [corresponding `ClientState` implementation](./client-state.md).

## `GetTimestamp` method

`GetTimestamp` should return the timestamp (in nanoseconds) of the consensus state snapshot.

## `ValidateBasic` method

`ValidateBasic` should validate every consensus state field and should return an error if any value is invalid. The light client implementer is in charge of determining which checks are required.


# Implementing the `ClientMessage` interface

As mentioned above, [`ClientMessage`](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L145) is an interface used to update an IBC client. This update may be done by a single header, a batch of headers, misbehaviour, or any type which when verified produces a change to the consensus state of the IBC client. This interface has been purposefully kept generic in order to give the maximum amount of flexibility to the light client implementer. 

```golang 
type ClientMessage interface {
	proto.Message

	ClientType() string
	ValidateBasic() error
}
```

The `ClientMessage` will be passed to the client to be used in [`UpdateClient`](https://github.com/cosmos/ibc-go/blob/57da75a70145409247e85365b64a4b2fc6ddad2f/modules/core/02-client/keeper/client.go#L53), which will handle a number of cases including misbehaviour and/or updating the consensus state. However, this `UpdateClient` function will always reference the specific functions determined by the relevant `ClientState`. This is because `UpdateClient` retrieves the client state by client ID (available in `MsgUpdateClient`). This client state implements the `ClientState` interface for a specific client type (e.g. Tendermint). The functions called on the client state instance in `UpdateClient` will be the specific implementations of `VerifyClientMessage`, `CheckForMisbehaviour`, `UpdateStateOnMisbehaviour` and `UpdateState` functions of the `ClientState` interface for that particular client type.