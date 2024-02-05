package celestia

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// CheckSubstituteAndUpdateState implements exported.ClientState.
func (*ClientState) CheckSubstituteAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore storetypes.KVStore, substituteClientStore storetypes.KVStore, substituteClient exported.ClientState) error {
	panic("unimplemented")
}

// VerifyUpgradeAndUpdateState implements exported.ClientState.
func (*ClientState) VerifyUpgradeAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, store storetypes.KVStore, newClient exported.ClientState, newConsState exported.ConsensusState, upgradeClientProof []byte, upgradeConsensusStateProof []byte) error {
	panic("unimplemented")
}

// ZeroCustomFields implements exported.ClientState.
func (*ClientState) ZeroCustomFields() exported.ClientState {
	panic("unimplemented")
}
