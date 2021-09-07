package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	parenttypes "github.com/cosmos/ibc-go/modules/apps/ccv/parent/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, genState types.ParentGenesisState) {
	k.SetPort(ctx, parenttypes.PortID)

	// Only try to bind to port if it is not already bound, since we may already own
	// port capability from capability InitGenesis
	if !k.IsBound(ctx, parenttypes.PortID) {
		// transfer module binds to the transfer port on InitChain
		// and claims the returned capability
		err := k.BindPort(ctx, parenttypes.PortID)
		if err != nil {
			panic(fmt.Sprintf("could not claim port capability: %v", err))
		}
	}

	// Set initial state for each child chain
	for _, cc := range genState.ChildStates {
		k.SetChainToChannel(ctx, cc.ChainId, cc.ChannelId)
		k.SetChannelToChain(ctx, cc.ChannelId, cc.ChainId)
		k.SetChannelStatus(ctx, cc.ChannelId, cc.Status)
	}
}
