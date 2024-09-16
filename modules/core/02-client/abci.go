package client

import (
	"context"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
)

// BeginBlocker is used to perform IBC client upgrades
func BeginBlocker(ctx context.Context, k *keeper.Keeper) {
	plan, err := k.GetUpgradePlan(ctx)
	if err == nil {
		// Once we are at the last block this chain will commit, set the upgraded consensus state
		// so that IBC clients can use the last NextValidatorsHash as a trusted kernel for verifying
		// headers on the next version of the chain.
		// Set the time to the last block time of the current chain.
		// In order for a client to upgrade successfully, the first block of the new chain must be committed
		// within the trusting period of the last block time on this chain.
		_, err := k.GetUpgradedClient(ctx, plan.Height)
		hi := k.HeaderService.HeaderInfo(ctx)
		if err == nil && hi.Height == plan.Height-1 {
			upgradedConsState := &ibctm.ConsensusState{
				Timestamp: hi.Time,
				// NextValidatorsHash: hi.NextValidatorsHash, //TODO: need to pass the consensus modules blocked on https://github.com/cosmos/cosmos-sdk/pull/21480
			}
			bz := types.MustMarshalConsensusState(k.Codec(), upgradedConsState)

			// SetUpgradedConsensusState always returns nil, hence the blank here.
			_ = k.SetUpgradedConsensusState(ctx, plan.Height, bz)

			keeper.EmitUpgradeChainEvent(ctx, k.Environment, plan.Height)
		}
	}
}
