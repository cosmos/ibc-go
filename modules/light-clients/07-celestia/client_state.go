package celestia

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

// ClientType implements exported.ClientState.
func (*ClientState) ClientType() string {
	return ModuleName
}

// GetLatestHeight implements exported.ClientState.
func (cs *ClientState) GetLatestHeight() exported.Height {
	return cs.BaseClient.GetLatestHeight()
}

// GetTimestampAtHeight implements exported.ClientState.
func (cs *ClientState) GetTimestampAtHeight(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, height exported.Height) (uint64, error) {
	return cs.BaseClient.GetTimestampAtHeight(ctx, clientStore, cdc, height)
}

// Status implements exported.ClientState.
func (cs *ClientState) Status(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec) exported.Status {
	return cs.BaseClient.Status(ctx, clientStore, cdc)
}

// Initialize implements exported.ClientState.
func (*ClientState) Initialize(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, consensusState exported.ConsensusState) error {
	panic("unimplemented")
}

// Validate implements exported.ClientState.
func (*ClientState) Validate() error {
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
