/*
Package solomachine implements a concrete LightClientModule, ClientState, ConsensusState,
Header and Misbehaviour types for the Solo Machine light client.
This implementation is based off the ICS 06 specification
(https://github.com/cosmos/ibc/tree/master/spec/client/ics-006-solo-machine-client)

CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to have the format 06-solomachine-{n}.
*/
package solomachine
