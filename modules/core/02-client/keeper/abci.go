package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

func (k *Keeper) ClientUpgrades(ctx sdk.Context) {
	plan, err := k.GetUpgradePlan(ctx)
	if err == nil {
		// Once we are at the last block this chain will commit, set the upgraded consensus state
		// so that IBC clients can use the last NextValidatorsHash as a trusted kernel for verifying
		// headers on the next version of the chain.
		// Set the time to the last block time of the current chain.
		// In order for a client to upgrade successfully, the first block of the new chain must be committed
		// within the trusting period of the last block time on this chain.
		_, err := k.GetUpgradedClient(ctx, plan.Height)
		if err == nil && ctx.BlockHeight() == plan.Height-1 {
			upgradedConsState := &ibctm.ConsensusState{
				Timestamp:          ctx.BlockTime(),
				NextValidatorsHash: ctx.BlockHeader().NextValidatorsHash,
			}
			bz := types.MustMarshalConsensusState(k.cdc, upgradedConsState)

			// SetUpgradedConsensusState always returns nil, hence the blank here.
			_ = k.SetUpgradedConsensusState(ctx, plan.Height, bz)

			EmitUpgradeChainEvent(ctx, plan.Height)
		}
	}

	// update the localhost client with the latest block height if it is active.
	if clientState, found := k.GetClientState(ctx, exported.Localhost); found {
		if k.GetClientStatus(ctx, exported.LocalhostClientID) == exported.Active {
			k.UpdateLocalhostClient(ctx, clientState)
		}
	}
}
