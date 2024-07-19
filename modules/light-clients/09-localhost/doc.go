/*
Package solomachine implements a concrete LightClientModule, ClientState, ConsensusState,
Header and Misbehaviour types for the Localhost light client.
This implementation is based off the ICS 09 specification
(https://github.com/cosmos/ibc/blob/main/spec/client/ics-009-loopback-cilent)
*/
// CONTRACT: clientID is validated in 02-client router, thus clientID is assumed here to be 09-localhost.

package localhost
