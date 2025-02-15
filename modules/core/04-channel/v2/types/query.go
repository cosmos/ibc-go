package types

import (
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

// NewQueryNextSequenceSendRequest creates a new next sequence send query.
func NewQueryNextSequenceSendRequest(clientID string) *QueryNextSequenceSendRequest {
	return &QueryNextSequenceSendRequest{
		ClientId: clientID,
	}
}

// NewQueryNextSequenceSendResponse creates a new QueryNextSequenceSendResponse instance
func NewQueryNextSequenceSendResponse(
	sequence uint64, proof []byte, height clienttypes.Height,
) *QueryNextSequenceSendResponse {
	return &QueryNextSequenceSendResponse{
		NextSequenceSend: sequence,
		Proof:            proof,
		ProofHeight:      height,
	}
}

// NewQueryPacketCommitmentRequest creates and returns a new packet commitment query request.
func NewQueryPacketCommitmentRequest(clientID string, sequence uint64) *QueryPacketCommitmentRequest {
	return &QueryPacketCommitmentRequest{
		ClientId: clientID,
		Sequence: sequence,
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

// NewQueryPacketAcknowledgementRequest creates and returns a new packet acknowledgement query request.
func NewQueryPacketAcknowledgementRequest(clientID string, sequence uint64) *QueryPacketAcknowledgementRequest {
	return &QueryPacketAcknowledgementRequest{
		ClientId: clientID,
		Sequence: sequence,
	}
}

// NewQueryPacketAcknowledgementResponse creates and returns a new packet acknowledgement query response.
func NewQueryPacketAcknowledgementResponse(acknowledgementHash []byte, proof []byte, proofHeight clienttypes.Height) *QueryPacketAcknowledgementResponse {
	return &QueryPacketAcknowledgementResponse{
		Acknowledgement: acknowledgementHash,
		Proof:           proof,
		ProofHeight:     proofHeight,
	}
}

// NewQueryPacketReceiptRequest creates and returns a new packet receipt query request.
func NewQueryPacketReceiptRequest(clientID string, sequence uint64) *QueryPacketReceiptRequest {
	return &QueryPacketReceiptRequest{
		ClientId: clientID,
		Sequence: sequence,
	}
}

// NewQueryPacketReceiptResponse creates and returns a new packet receipt query response.
func NewQueryPacketReceiptResponse(exists bool, proof []byte, height clienttypes.Height) *QueryPacketReceiptResponse {
	return &QueryPacketReceiptResponse{
		Received:    exists,
		Proof:       proof,
		ProofHeight: height,
	}
}

// NewQueryPacketReceiptRequest creates and returns a new packet receipt query request.
func NewQueryUnreceivedPacketsRequest(clientID string, sequences []uint64) *QueryUnreceivedPacketsRequest {
	return &QueryUnreceivedPacketsRequest{
		ClientId:  clientID,
		Sequences: sequences,
	}
}
