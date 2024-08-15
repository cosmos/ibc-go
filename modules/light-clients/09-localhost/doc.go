/*
Package solomachine implements a concrete LightClientModule, ClientState, ConsensusState,
Header and Misbehaviour types for the Localhost light client.
This implementation is based off the ICS 09 specification
(https://github.com/cosmos/ibc/blob/main/spec/client/ics-009-loopback-cilent)

Note the client identifier is expected to be: 09-localhost.
This is validated by core IBC in the 02-client submodule.
*/
package localhost
