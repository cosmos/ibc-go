package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ clienttypes.SelfClientValidator = (*MockClientValidator)(nil)

type MockClientValidator struct {
	GetSelfConsensusStateFn func(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error)
	ValidateSelfClientFn    func(ctx sdk.Context, clientState exported.ClientState) error
}

func (mcv *MockClientValidator) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
	if mcv.GetSelfConsensusStateFn == nil {
		return nil, nil
	}

	return mcv.GetSelfConsensusStateFn(ctx, height)
}

func (mcv *MockClientValidator) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	if mcv.ValidateSelfClientFn == nil {
		return nil
	}

	return mcv.ValidateSelfClientFn(ctx, clientState)
}
