package types

import (
	"context"

	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type ClientRouter interface {
	// Route returns the client module for the given client ID
	Route(clientID string) exported.LightClientModule
}

type IBCLiteKeeper interface {
	// GetCounterparty returns the counterparty client given the client ID on
	// the executing chain
	// This is a private path that is only used by the IBC lite module
	GetCounterparty(ctx context.Context, clientID string) (counterpartyClientID string)

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

	// SetPacketAcknowledgement writes the acknowledgement hash under the acknowledgement path
	// This is a public path that is standardized by the IBC specification
	SetPacketAcknowledgement(ctx context.Context, portID, channelID string, sequence uint64, ackHash []byte)
}

type AppRouter interface {
	Route(portID string) porttypes.IBCModule
}
