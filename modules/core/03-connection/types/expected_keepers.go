package types

import (
	"context"

	paramtypes "cosmossdk.io/x/params/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// ClientKeeper expected account IBC client keeper
type ClientKeeper interface {
	GetClientStatus(ctx context.Context, clientID string) exported.Status
	GetClientState(ctx context.Context, clientID string) (exported.ClientState, bool)
	GetClientConsensusState(ctx context.Context, clientID string, height exported.Height) (exported.ConsensusState, bool)
	VerifyMembership(ctx context.Context, clientID string, height exported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path exported.Path, value []byte) error
	VerifyNonMembership(ctx context.Context, clientID string, height exported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path exported.Path) error
	IterateClientStates(ctx context.Context, prefix []byte, cb func(string, exported.ClientState) bool)
}

// ParamSubspace defines the expected Subspace interface for module parameters.
type ParamSubspace interface {
	GetParamSet(ctx sdk.Context, ps paramtypes.ParamSet)
}
