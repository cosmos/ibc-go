package cli

import (
	"context"
	"encoding/binary"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/client"

	clientutils "github.com/cosmos/ibc-go/v9/modules/core/02-client/client/utils"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	ibcclient "github.com/cosmos/ibc-go/v9/modules/core/client"
)

func queryChannelClientStateABCI(clientCtx client.Context, channelID string) (*types.QueryChannelClientStateResponse, error) {
	queryClient := types.NewQueryClient(clientCtx)
	req := &types.QueryChannelClientStateRequest{
		ChannelId: channelID,
	}

	res, err := queryClient.ChannelClientState(context.Background(), req)
	if err != nil {
		return nil, err
	}

	clientStateRes, err := clientutils.QueryClientStateABCI(clientCtx, res.IdentifiedClientState.ClientId)
	if err != nil {
		return nil, err
	}

	// use client state returned from ABCI query in case query height differs
	identifiedClientState := clienttypes.IdentifiedClientState{
		ClientId:    res.IdentifiedClientState.ClientId,
		ClientState: clientStateRes.ClientState,
	}
	res = types.NewQueryChannelClientStateResponse(identifiedClientState, clientStateRes.Proof, clientStateRes.ProofHeight)

	return res, nil
}

func queryNextSequenceSendABCI(clientCtx client.Context, channelID string) (*types.QueryNextSequenceSendResponse, error) {
	key := hostv2.NextSequenceSendKey(channelID)
	value, proofBz, proofHeight, err := ibcclient.QueryTendermintProof(clientCtx, key)
	if err != nil {
		return nil, err
	}

	// check if next sequence send exists
	if len(value) == 0 {
		return nil, errorsmod.Wrapf(types.ErrSequenceSendNotFound, "channelID (%s)", channelID)
	}

	sequence := binary.BigEndian.Uint64(value)

	return types.NewQueryNextSequenceSendResponse(sequence, proofBz, proofHeight), nil
}

func queryPacketCommitmentABCI(clientCtx client.Context, channelID string, sequence uint64) (*types.QueryPacketCommitmentResponse, error) {
	key := hostv2.PacketCommitmentKey(channelID, sequence)
	value, proofBz, proofHeight, err := ibcclient.QueryTendermintProof(clientCtx, key)
	if err != nil {
		return nil, err
	}

	// check if packet commitment exists
	if len(value) == 0 {
		return nil, errorsmod.Wrapf(types.ErrPacketCommitmentNotFound, "channelID (%s), sequence (%d)", channelID, sequence)
	}

	return types.NewQueryPacketCommitmentResponse(value, proofBz, proofHeight), nil
}

func queryPacketAcknowledgementABCI(clientCtx client.Context, channelID string, sequence uint64) (*types.QueryPacketAcknowledgementResponse, error) {
	key := hostv2.PacketAcknowledgementKey(channelID, sequence)
	value, proofBz, proofHeight, err := ibcclient.QueryTendermintProof(clientCtx, key)
	if err != nil {
		return nil, err
	}

	// check if packet commitment exists
	if len(value) == 0 {
		return nil, errorsmod.Wrapf(types.ErrAcknowledgementNotFound, "channelID (%s), sequence (%d)", channelID, sequence)
	}

	return types.NewQueryPacketAcknowledgementResponse(value, proofBz, proofHeight), nil
}

func queryPacketReceiptABCI(clientCtx client.Context, channelID string, sequence uint64) (*types.QueryPacketReceiptResponse, error) {
	key := hostv2.PacketReceiptKey(channelID, sequence)

	value, proofBz, proofHeight, err := ibcclient.QueryTendermintProof(clientCtx, key)
	if err != nil {
		return nil, err
	}

	return types.NewQueryPacketReceiptResponse(value != nil, proofBz, proofHeight), nil
}
