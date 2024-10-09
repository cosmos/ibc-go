package types

import (
	"context"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

type ChannelKeeper interface {
	// SetPacketCommitment writes the commitment hash under the commitment path
	// This is a public path that is standardized by the IBC specification
	SetPacketCommitment(ctx context.Context, portID string, channelID string, sequence uint64, commitment []byte)

	// GetPacketCommitment returns the packet commitment hash under the commitment path
	GetPacketCommitment(ctx context.Context, portID string, channelID string, sequence uint64) []byte

	// DeletePacketCommitment deletes the packet commitment hash under the commitment path
	DeletePacketCommitment(ctx context.Context, portID string, channelID string, sequence uint64)

	// SetNextSequenceSend writes the next send sequence under the sequence path
	// This is a public path that is standardized by the IBC specification
	SetNextSequenceSend(ctx context.Context, portID, channelID string, sequence uint64)

	// GetNextSequenceSend returns the next send sequence from the sequence path
	GetNextSequenceSend(ctx context.Context, portID, channelID string) (uint64, bool)

	// SetPacketReceipt writes the packet receipt under the receipt path
	// This is a public path that is standardized by the IBC specification
	SetPacketReceipt(ctx context.Context, portID, channelID string, sequence uint64)

	// GetPacketReceipt returns the packet receipt from the packet receipt path
	GetPacketReceipt(ctx context.Context, portID, channelID string, sequence uint64) (string, bool)

	// HasPacketAcknowledgement check if the packet ack hash is already on the store
	HasPacketAcknowledgement(ctx context.Context, portID, channelID string, sequence uint64) bool

	// SetPacketAcknowledgement writes the acknowledgement hash under the acknowledgement path
	// This is a public path that is standardized by the IBC specification
	SetPacketAcknowledgement(ctx context.Context, portID, channelID string, sequence uint64, ackHash []byte)

	// GetV2Counterparty returns a version 2 counterparty for a given portID and channel ID
	GetV2Counterparty(ctx context.Context, portID, channelID string) (Counterparty, bool)
}

type ClientKeeper interface {
	// VerifyMembership retrieves the light client module for the clientID and verifies the proof of the existence of a key-value pair at a specified height.
	VerifyMembership(ctx context.Context, clientID string, height exported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path exported.Path, value []byte) error
	// VerifyNonMembership retrieves the light client module for the clientID and verifies the absence of a given key at a specified height.
	VerifyNonMembership(ctx context.Context, clientID string, height exported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path exported.Path) error
	// GetClientStatus returns the status of a client given the client ID
	GetClientStatus(ctx context.Context, clientID string) exported.Status
	// GetClientLatestHeight returns the latest height of a client given the client ID
	GetClientLatestHeight(ctx context.Context, clientID string) clienttypes.Height
	// GetClientTimestampAtHeight returns the timestamp for a given height on the client
	// given its client ID and height
	GetClientTimestampAtHeight(ctx context.Context, clientID string, height exported.Height) (uint64, error)
}
