package cli

import (
	"encoding/binary"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	ibcclient "github.com/cosmos/ibc-go/v9/modules/core/client"
)

func queryNextSequenceSendABCI(clientCtx client.Context, channelID string) (*types.QueryNextSequenceSendResponse, error) {
	key := host.NextSequenceSendKey(channelID)
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
	key := host.PacketCommitmentKey(channelID, sequence)
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
	key := host.PacketAcknowledgementKey(channelID, sequence)
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
	key := host.PacketReceiptKey(channelID, sequence)

	value, proofBz, proofHeight, err := ibcclient.QueryTendermintProof(clientCtx, key)
	if err != nil {
		return nil, err
	}

	return types.NewQueryPacketReceiptResponse(value != nil, proofBz, proofHeight), nil
}
