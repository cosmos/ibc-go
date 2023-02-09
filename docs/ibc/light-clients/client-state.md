<!--
order: 2
-->

# Implementing the `ClientState` interface

Learn how to implement the [`ClientState`](https://github.com/cosmos/ibc-go/blob/v6.0.0/modules/core/exported/client.go#L40) interface. This list of methods described here does not include all methods of the interface. Some methods are explained in detail in the relevant sections of the guide.

## `ClientType` method

`ClientType` should return a unique string identifier of the light client. This will be used when generating a client identifier.
The format is created as follows: `ClientType-{N}` where `{N}` is the unique global nonce associated with a specific client.

## `GetLatestHeight` method

`GetLatestHeight` should return the latest block height that the client state represents.

## `Validate` method

`Validate` should validate every client state field and should return an error if any value is invalid. The light client
implementer is in charge of determining which checks are required. See the [Tendermint light client implementation](https://github.com/cosmos/ibc-go/blob/v6.0.0/modules/light-clients/07-tendermint/types/client_state.go#L101) as a reference.

## `Status` method

`Status` must return the status of the client.

- An `Active` status indicates that clients are allowed to process packets.
- A `Frozen` status indicates that misbehaviour was detected in the counterparty chain and the client is not allowed to be used.
- An `Expired` status indicates that a client is not allowed to be used because it was not updated for longer than the trusting period.
- An `Unknown` status indicates that there was an error in determining the status of a client.

All possible `Status` types can be found [here](https://github.com/cosmos/ibc-go/blob/v6.0.0/modules/core/exported/client.go#L26-L36).

This field is returned in the response of the gRPC [`ibc.core.client.v1.Query/ClientStatus`](https://github.com/cosmos/ibc-go/blob/v6.0.0/modules/core/02-client/types/query.pb.go#L665) endpoint.

## `ZeroCustomFields` method

`ZeroCustomFields` should return a copy of the light client with all client customizable fields with their zero value. It should not mutate the fields of the light client.
This method is used when [scheduling upgrades](https://github.com/cosmos/ibc-go/blob/v6.0.0/modules/core/02-client/keeper/proposal.go#L89). Upgrades are used to upgrade chain specific fields. 
In the tendermint case, this may be the chain ID or the unbonding period.
For more information about client upgrades see the [Handling upgrades](./upgrades.md) section.

## `GetTimestampAtHeight` method

`GetTimestampAtHeight` must return the timestamp for the consensus state associated with the provided height.
This value is used to facilitate timeouts by checking the packet timeout timestamp against the returned value.

## `Initialize` method

Clients must validate the initial consensus state, and set the initial client state and consensus state in the provided client store.
Clients may also store any necessary client-specific metadata.

`Initialize` is called when a [client is created](https://github.com/cosmos/ibc-go/blob/main/modules/core/02-client/keeper/client.go#L32).

## `VerifyMembership` method

`VerifyMembership` must verify the existence of a value at a given commitment path at the specified height. For more information about membership proofs
see the [Existence and non-existence proofs section](./proofs.md).

## `VerifyNonMembership` method

`VerifyNonMembership` must verify the absence of a value at a given commitment path at a specified height. For more information about non-membership proofs
see the [Existence and non-existence proofs section](./proofs.md).

## `VerifyClientMessage` method

`VerifyClientMessage` must verify a `ClientMessage`. A `ClientMessage` could be a `Header`, `Misbehaviour`, or batch update.
It must handle each type of `ClientMessage` appropriately. Calls to `CheckForMisbehaviour`, `UpdateState`, and `UpdateStateOnMisbehaviour`
will assume that the content of the `ClientMessage` has been verified and can be trusted. An error should be returned
if the ClientMessage fails to verify.

## `CheckForMisbehaviour` method

Checks for evidence of a misbehaviour in `Header` or `Misbehaviour` type. It assumes the `ClientMessage`
has already been verified.
