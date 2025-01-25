package client

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cometabci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
)

// BeginBlocker is used to perform IBC client upgrades.
// Uses comet service through blockHeader to access full block header information,
// including NextValidatorsHash and other header data.
func BeginBlocker(goCtx context.Context, k *keeper.Keeper, blockHeader *cometabci.RequestBeginBlock) {
	ctx := sdk.UnwrapSDKContext(goCtx)

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
				Timestamp:          time.Unix(0, blockHeader.Header.Time.UnixNano()),
				NextValidatorsHash: blockHeader.Header.NextValidatorsHash,
			}
			bz := types.MustMarshalConsensusState(k.Codec(), upgradedConsState)

			// SetUpgradedConsensusState always returns nil, hence the blank here.
			_ = k.SetUpgradedConsensusState(ctx, plan.Height, bz)

			if err := k.EmitUpgradeChainEvent(ctx, plan.Height); err != nil {
				k.Logger.Error("error in events emission", "error", err.Error())
			}
		}
	}
}
