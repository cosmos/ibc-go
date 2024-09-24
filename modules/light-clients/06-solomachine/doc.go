/*
Package solomachine implements a concrete LightClientModule, ClientState, ConsensusState,
Header and Misbehaviour types for the Solo Machine light client.
This implementation is based off the ICS 06 specification
(https://github.com/cosmos/ibc/tree/master/spec/client/ics-006-solo-machine-client)

Note that client identifiers are expected to be in the form: 06-solomachine-{N}.
Client identifiers are generated and validated by core IBC, unexpected client identifiers will result in errors.
*/
package solomachine
