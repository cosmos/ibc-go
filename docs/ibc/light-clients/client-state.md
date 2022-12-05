<!--
order: 2
-->

# Implementing the ClientState interface

Learn how to implement the [Client State](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L36) interface.

### ClientType

`ClientType` should return a unique string identifier of the light client.

### GetLatestHeight

`GetLatestHeight` should return the latest block height.

### Validate

`Validate` should validate every client state field and should return an error if any value is invalid. The light client
implementor is in charge of determining which checks are required. See the [tendermint light client implementation](https://github.com/cosmos/ibc-go/blob/main/modules/light-clients/07-tendermint/client_state.go#L111)
as a reference.

### Status

`Status` must return the status of the client. Only `Active` clients are allowed to process packets. All
possible Status types can be found [here](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go).

### ZeroCustomFields

`ZeroCustomFields` should return a copy of the light client with all client customizable fields with their zero value. It should not mutate the fields of the light client.
This method is used to [verify upgrades](https://github.com/cosmos/ibc-go/blob/main/modules/core/02-client/types/proposal.go#L120) and when [scheduling upgrades](https://github.com/cosmos/ibc-go/blob/main/modules/core/02-client/keeper/proposal.go#L82).

### GetTimestampAtHeight

`GetTimestampAtHeight` must return the timestamp for the consensus state associated with the provided height.

### Initialize

Clients must validate the initial consensus state, and may store any client-specific metadata necessary
for correct light client operations in the `Initialize` function.

`Initialize` gets called when a [client is created](https://github.com/cosmos/ibc-go/blob/main/modules/core/02-client/keeper/client.go#L32).

### VerifyMembership

`VerifyMembership` must verify the existence of a value at a given CommitmentPath at the specified height.
The caller of this function is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized
path (as defined in ICS 24).

### VerifyNonMembership

`VerifyNonMembership` must verify the absence of a given CommitmentPath at a specified height.
The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).

### VerifyClientMessage

VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
if the ClientMessage fails to verify.

### CheckForMisbehaviour

Checks for evidence of a misbehaviour in Header or Misbehaviour type. It assumes the ClientMessage
has already been verified.
