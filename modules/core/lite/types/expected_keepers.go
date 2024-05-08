package types

import (
	"context"

	port "github.com/cosmos/ibc-go/v8/modules/core/05-port"
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
	GetCounterparty(ctx context.Context, portID, channelID string) (portIDOut, channelIDOut string, found bool)

	// WriteAcknowledgement writes the acknowledgement under the acknowledgement path
	WriteAcknowledgement(ctx context.Context, portID, channelID string, acknowledgement []byte) error
}

type AppRouter interface {
	Route(portID string) port.AppModule
}
