package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// InitGenesis sets the genesis state in the store.
func (k *Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	// set acks
	for _, ack := range data.Acknowledgements {
		k.SetPacketAcknowledgement(ctx, ack.ClientId, ack.Sequence, ack.Data)
	}

	// set commits
	for _, commitment := range data.Commitments {
		k.SetPacketCommitment(ctx, commitment.ClientId, commitment.Sequence, commitment.Data)
	}

	// set receipts
	for _, receipt := range data.Receipts {
		k.SetPacketReceipt(ctx, receipt.ClientId, receipt.Sequence)
	}

	// set send sequences
	for _, seq := range data.SendSequences {
		k.SetNextSequenceSend(ctx, seq.ClientId, seq.Sequence)
	}
}

// ExportGenesis exports the current state to a genesis state.
func (k *Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	clientStates := k.ClientKeeper.GetAllGenesisClients(ctx)
	gs := types.GenesisState{
		Acknowledgements: make([]types.PacketState, 0),
		Commitments:      make([]types.PacketState, 0),
		Receipts:         make([]types.PacketState, 0),
		SendSequences:    make([]types.PacketSequence, 0),
	}
	for _, clientState := range clientStates {
		acks := k.GetAllPacketAcknowledgementsForClient(ctx, clientState.ClientId)
		gs.Acknowledgements = append(gs.Acknowledgements, acks...)

		comms := k.GetAllPacketCommitmentsForClient(ctx, clientState.ClientId)
		gs.Commitments = append(gs.Commitments, comms...)

		receipts := k.GetAllPacketReceiptsForClient(ctx, clientState.ClientId)
		gs.Receipts = append(gs.Receipts, receipts...)

		seq, ok := k.GetNextSequenceSend(ctx, clientState.ClientId)
		if ok {
			gs.SendSequences = append(gs.SendSequences, types.NewPacketSequence(clientState.ClientId, seq))
		}
	}

	return gs
}
