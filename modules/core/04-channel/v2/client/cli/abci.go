package cli

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	ibcclient "github.com/cosmos/ibc-go/v9/modules/core/client"
)

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
