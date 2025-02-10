package channelv2

import (
	"context"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

func InitGenesis(ctx context.Context, k *keeper.Keeper, gs types.GenesisState) {
	// set acks
	for _, ack := range gs.Acknowledgements {
		k.SetPacketAcknowledgement(ctx, ack.ClientId, ack.Sequence, ack.Data)
	}

	// set commits
	for _, commitment := range gs.Commitments {
		k.SetPacketCommitment(ctx, commitment.ClientId, commitment.Sequence, commitment.Data)
	}

	// set receipts
	for _, receipt := range gs.Receipts {
		k.SetPacketReceipt(ctx, receipt.ClientId, receipt.Sequence)
	}

	// set send sequences
	for _, seq := range gs.SendSequences {
		k.SetNextSequenceSend(ctx, seq.ClientId, seq.Sequence)
	}
}

func ExportGenesis(ctx context.Context, k *keeper.Keeper) types.GenesisState {
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
