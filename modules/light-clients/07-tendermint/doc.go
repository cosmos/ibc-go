/*
Package tendermint implements a concrete LightClientModule, ClientState, ConsensusState,
Header, Misbehaviour and types for the Tendermint consensus light client.
This implementation is based off the ICS 07 specification
(https://github.com/cosmos/ibc/tree/main/spec/client/ics-007-tendermint-client)
*/
package tendermint

// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 07-tendermint-{n}.
