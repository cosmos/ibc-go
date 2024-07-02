package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ clienttypes.ConsensusHost = (*ConsensusHost)(nil)

type ConsensusHost struct {
	GetSelfConsensusStateFn func(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error)
	ValidateSelfClientFn    func(ctx sdk.Context, clientState exported.ClientState) error
}

func (cv *ConsensusHost) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
	if cv.GetSelfConsensusStateFn == nil {
		return nil, nil
	}

	return cv.GetSelfConsensusStateFn(ctx, height)
}

func (cv *ConsensusHost) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	if cv.ValidateSelfClientFn == nil {
		return nil
	}

	return cv.ValidateSelfClientFn(ctx, clientState)
}
