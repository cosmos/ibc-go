package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/child/types"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
)

// InitGenesis initializes the CCV child state and binds to PortID.
func (k Keeper) InitGenesis(ctx sdk.Context, state ccv.ChildGenesisState) {
	k.SetPort(ctx, types.PortID)

	// Only try to bind to port if it is not already bound, since we may already own
	// port capability from capability InitGenesis
	if !k.IsBound(ctx, types.PortID) {
		// transfer module binds to the transfer port on InitChain
		// and claims the returned capability
		err := k.BindPort(ctx, types.PortID)
		if err != nil {
			panic(fmt.Sprintf("could not claim port capability: %v", err))
		}
	}

	// set parent chain id.
	k.SetParentChain(ctx, state.ParentChainId)
	if state.NewChain {
		// Create the parent client in InitGenesis for new child chain. CCV Handshake must be established with this client id.
		clientID, err := k.clientKeeper.CreateClient(ctx, state.ParentClientState, state.ParentConsensusState)
		panic(err)
		// set parent client id.
		k.SetParentClient(ctx, clientID)
	} else {
		// set parent channel id.
		k.SetParentChannel(ctx, state.ParentChannelId)
		// set all unbonding sequences
		for _, us := range state.UnbondingSequences {
			k.SetUnbondingTime(ctx, us.Sequence, us.UnbondingTime)
			k.SetUnbondingPacket(ctx, us.Sequence, *us.UnbondingPacket)
		}
	}
}
