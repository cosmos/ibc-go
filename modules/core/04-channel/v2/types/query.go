package types

import clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"

// NewQueryChannelRequest creates and returns a new channel query request.
func NewQueryChannelRequest(channelID string) *QueryChannelRequest {
	return &QueryChannelRequest{
		ChannelId: channelID,
	}
}

// NewQueryChannelResponse creates and returns a new channel query response.
func NewQueryChannelResponse(creator string, channel Channel) *QueryChannelResponse {
	return &QueryChannelResponse{
		Creator: creator,
		Channel: channel,
	}
}

// NewQueryPacketCommitmentRequest creates and returns a new packet commitment query request.
func NewQueryPacketCommitmentRequest(channelID string, sequence uint64) *QueryPacketCommitmentRequest {
	return &QueryPacketCommitmentRequest{
		ChannelId: channelID,
		Sequence:  sequence,
	}
}

// NewQueryPacketCommitmentResponse creates and returns a new packet commitment query response.
func NewQueryPacketCommitmentResponse(commitmentHash []byte, proof []byte, proofHeight clienttypes.Height) *QueryPacketCommitmentResponse {
	return &QueryPacketCommitmentResponse{
		Commitment:  commitmentHash,
		Proof:       proof,
		ProofHeight: proofHeight,
	}
}
