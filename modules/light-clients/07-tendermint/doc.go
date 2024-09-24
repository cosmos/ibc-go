/*
Package tendermint implements a concrete LightClientModule, ClientState, ConsensusState,
Header, Misbehaviour and types for the Tendermint consensus light client.
This implementation is based off the ICS 07 specification
(https://github.com/cosmos/ibc/tree/main/spec/client/ics-007-tendermint-client)

Note that client identifiers are expected to be in the form: 07-tendermint-{N}.
Client identifiers are generated and validated by core IBC, unexpected client identifiers will result in errors.
*/
package tendermint
