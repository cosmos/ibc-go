package celestia

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

// CheckForMisbehaviour implements exported.ClientState.
func (*ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) bool {
	panic("unimplemented")
}

// CheckSubstituteAndUpdateState implements exported.ClientState.
func (*ClientState) CheckSubstituteAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore storetypes.KVStore, substituteClientStore storetypes.KVStore, substituteClient exported.ClientState) error {
	panic("unimplemented")
}

// ClientType implements exported.ClientState.
func (*ClientState) ClientType() string {
	panic("unimplemented")
}

// ExportMetadata implements exported.ClientState.
func (*ClientState) ExportMetadata(clientStore storetypes.KVStore) []exported.GenesisMetadata {
	panic("unimplemented")
}

// GetLatestHeight implements exported.ClientState.
func (*ClientState) GetLatestHeight() exported.Height {
	panic("unimplemented")
}

// GetTimestampAtHeight implements exported.ClientState.
func (*ClientState) GetTimestampAtHeight(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, height exported.Height) (uint64, error) {
	panic("unimplemented")
}

// Initialize implements exported.ClientState.
func (*ClientState) Initialize(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, consensusState exported.ConsensusState) error {
	panic("unimplemented")
}

// Status implements exported.ClientState.
func (*ClientState) Status(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec) exported.Status {
	panic("unimplemented")
}

// UpdateState implements exported.ClientState.
func (*ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	panic("unimplemented")
}

// UpdateStateOnMisbehaviour implements exported.ClientState.
func (*ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) {
	panic("unimplemented")
}

// Validate implements exported.ClientState.
func (*ClientState) Validate() error {
	panic("unimplemented")
}

// VerifyClientMessage implements exported.ClientState.
func (*ClientState) VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) error {
	panic("unimplemented")
}

// VerifyMembership implements exported.ClientState.
func (*ClientState) VerifyMembership(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, height exported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path exported.Path, value []byte) error {
	panic("unimplemented")
}

// VerifyNonMembership implements exported.ClientState.
func (*ClientState) VerifyNonMembership(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, height exported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path exported.Path) error {
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
