package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type ClientRouter interface {
	// Route returns the client module for the given client ID
	Route(clientID string) exported.LightClientModule
}

type ChannelKeeper interface {
	// GetCounterparty returns the counterparty channelID given the channel ID on
	// the executing chain
	// Note for IBC lite, the portID is not needed as there is effectively
	// a single channel between the two clients that can switch between apps using the portID
	GetLiteCounterparty(ctx sdk.Context, portID, channelID string) (portIDOut, channelIDOut string, found bool)

	GetPacketCommitment(ctx sdk.Context, portID string, channelID string, sequence uint64) []byte

	DeletePacketCommitment(ctx sdk.Context, portID string, channelID string, sequence uint64)

	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)

	SetNextSequenceSend(ctx sdk.Context, portID, channelID string, sequence uint64)

	SetPacketCommitment(ctx sdk.Context, portID string, channelID string, sequence uint64, commitment []byte)

	// WriteAcknowledgement writes the acknowledgement under the acknowledgement path
	WriteAcknowledgement(ctx sdk.Context, portID, channelID string, acknowledgement []byte) error
}

type AppRouter interface {
	Route(portID string) porttypes.IBCModule
}
