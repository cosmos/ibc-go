package types

import (
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// verifyMisbehaviour determines whether or not two conflicting
// headers at the same height would have convinced the light client.
//
func (cs *ClientState) verifyMisbehaviour(ctx sdk.Context, clientStore sdk.KVStore, cdc codec.BinaryCodec, misbehaviour *Misbehaviour) error {
	panic("implement me")
}

// checkMisbehaviourHeader checks that a Header in Misbehaviour is valid misbehaviour given
// a trusted ConsensusState
func checkMisbehaviourHeader(
	clientState *ClientState, consState *ConsensusState, header *Header, currentTimestamp time.Time,
) error {
	panic("implement me")
}
