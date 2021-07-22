package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/child/types"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
)

// InitGenesis initializes the CCV child state and binds to PortID.
func (k Keeper) InitGenesis(ctx sdk.Context, state ccv.ChildGenesisState) {
	if state.Disabled {
		return
	}
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

	if state.NewChain {
		// Create the parent client in InitGenesis for new child chain. CCV Handshake must be established with this client id.
		clientID, err := k.clientKeeper.CreateClient(ctx, state.ParentClientState, state.ParentConsensusState)
		if err != nil {
			panic(err)
		}
		// set parent client id.
		k.SetParentClient(ctx, clientID)
	} else {
		// set parent channel id.
		k.SetParentChannel(ctx, state.ParentChannelId)
		// set all unbonding sequences
		for _, us := range state.UnbondingSequences {
			k.SetUnbondingTime(ctx, us.Sequence, us.UnbondingTime)
			k.SetUnbondingPacket(ctx, us.Sequence, us.UnbondingPacket)
		}
	}
}

// ExportGenesis exports the CCV child state. If the channel has already been established, then we export
// parent chain. Otherwise, this is still considered a new chain and we export latest client state.
func (k Keeper) ExportGenesis(ctx sdk.Context) ccv.ChildGenesisState {
	if channelID, ok := k.GetParentChannel(ctx); ok {
		gs := ccv.NewRestartChildGenesisState(channelID, nil)

		unbondingSequences := []ccv.UnbondingSequence{}
		cb := func(seq uint64, packet channeltypes.Packet) bool {
			timeNs := k.GetUnbondingTime(ctx, seq)
			us := ccv.UnbondingSequence{
				Sequence:        seq,
				UnbondingTime:   timeNs,
				UnbondingPacket: packet,
			}
			unbondingSequences = append(unbondingSequences, us)
			return false
		}
		k.IterateUnbondingPacket(ctx, cb)

		gs.UnbondingSequences = unbondingSequences
		return gs
	}
	clientID, ok := k.GetParentClient(ctx)
	// if parent clientID and channelID don't exist on the child chain, then CCV protocol is disabled for this chain
	// return a disabled genesis state
	if !ok {
		return ccv.DefaultChildGenesisState()
	}
	cs, ok := k.clientKeeper.GetClientState(ctx, clientID)
	if !ok {
		panic("parent client not set on already running child chain")
	}
	tmCs, ok := cs.(*ibctmtypes.ClientState)
	if !ok {
		panic("parent client consensus state is not tendermint client state")
	}
	consState, ok := k.clientKeeper.GetLatestClientConsensusState(ctx, clientID)
	if !ok {
		panic("parent consensus state not set on already running child chain")
	}
	tmConsState, ok := consState.(*ibctmtypes.ConsensusState)
	if !ok {
		panic("parent consensus state is not tendermint consensus state")
	}
	return ccv.NewInitialChildGenesisState(tmCs, tmConsState)
}
