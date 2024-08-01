package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
)
type ChannelKeeper interface {
	// SetPacketCommitment writes the commitment hash under the commitment path
	// This is a public path that is standardized by the IBC specification
	SetPacketCommitment(ctx sdk.Context, portID string, channelID string, sequence uint64, commitment []byte)

	// GetPacketCommitment returns the packet commitment hash under the commitment path
	GetPacketCommitment(ctx sdk.Context, portID string, channelID string, sequence uint64) []byte

	// DeletePacketCommitment deletes the packet commitment hash under the commitment path
	DeletePacketCommitment(ctx sdk.Context, portID string, channelID string, sequence uint64)

	// SetNextSequenceSend writes the next send sequence under the sequence path
	// This is a public path that is standardized by the IBC specification
	SetNextSequenceSend(ctx sdk.Context, portID, channelID string, sequence uint64)

	// GetNextSequenceSend returns the next send sequence from the sequence path
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)

	// SetPacketReceipt writes the packet receipt under the receipt path
	// This is a public path that is standardized by the IBC specification
	SetPacketReceipt(ctx sdk.Context, portID, channelID string, sequence uint64)

	// SetPacketAcknowledgement writes the acknowledgement hash under the acknowledgement path
	// This is a public path that is standardized by the IBC specification
	SetPacketAcknowledgement(ctx sdk.Context, portID, channelID string, sequence uint64, ackHash []byte)
}

type ClientKeeper interface {
	// GetCounterparty returns the counterparty client given the client ID on
	// the executing chain
	// This is a private path that is only used by the IBC lite module
	GetCounterparty(ctx sdk.Context, clientID string) (clienttypes.Counterparty, bool)
}
