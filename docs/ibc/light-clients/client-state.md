<!--
order: 2
-->

# Implementing the ClientState interface

Learn how to implement the [Client State](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go#L36) interface.

### ClientType

`ClientType() string` should return a unique string identifier of the light client.

### GetLatestHeight

`GetLatestHeight() Height` should return the latest block height.

### Validate

`Validate() error` should return an error if the given ClientState values are invalid.

### Status

`Status() Status` must return the status of the client. Only Active clients are allowed to process packets. All
possible Status types can be found [here](https://github.com/cosmos/ibc-go/blob/main/modules/core/exported/client.go).

### ZeroCustomFields

`ZeroCustomFields() ClientState` should zero out any client customizable fields in client state. Ledger enforced
fields are maintained while all custom fields are zero values, this is [used to verify upgrades](https://github.com/cosmos/ibc-go/blob/main/modules/core/02-client/types/proposal.go#L120).

### GetTimestampAtHeight

`GetTimestampAtHeight` must return the timestamp for the consensus state associated with the provided block height.

### Initialize

Clients must validate the initial consensus state, and may store any client-specific metadata necessary
for correct light client operations in the `Initialize` function.

`Initialize` gets called when a [client is created](https://github.com/cosmos/ibc-go/blob/main/modules/core/02-client/keeper/client.go#L32).

### VerifyMembership

`VerifyMembership` is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).

### VerifyNonMembership

`VerifyNonMembership` is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).

### VerifyClientMessage

VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
if the ClientMessage fails to verify.

### CheckForMisbehaviour

Checks for evidence of a misbehaviour in Header or Misbehaviour type. It assumes the ClientMessage
has already been verified.
