package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
)

// InitGenesis
func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {
	for _, fee := range state.IdentifiedFees {
		k.SetFeeInEscrow(ctx, fee)
	}

	for _, addr := range state.RegisteredRelayers {
		k.SetCounterpartyAddress(ctx, addr.Address, addr.CounterpartyAddress)
	}

	for _, forwardAddr := range state.ForwardRelayers {
		k.SetForwardRelayerAddress(ctx, forwardAddr.PacketId, forwardAddr.Address)
	}

	for _, enabledChan := range state.FeeEnabledChannels {
		k.SetFeeEnabled(ctx, enabledChan.PortId, enabledChan.ChannelId)
	}
}

// ExportGenesis
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		IdentifiedFees:     k.GetAllIdentifiedPacketFees(ctx),
		FeeEnabledChannels: k.GetAllFeeEnabledChannels(ctx),
		RegisteredRelayers: k.GetAllRelayerAddresses(ctx),
		ForwardRelayers:    k.GetAllForwardRelayerAddresses(ctx),
	}
}
