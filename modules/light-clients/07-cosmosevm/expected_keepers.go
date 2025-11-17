package cosmosevm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

type ClientKeeper interface {
	// VerifyMembership retrieves the light client module for the clientID and verifies the proof of the existence of a key-value pair at a specified height.
	VerifyMembership(ctx sdk.Context, clientID string, height ibcexported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path ibcexported.Path, value []byte) error
	// VerifyNonMembership retrieves the light client module for the clientID and verifies the absence of a given key at a specified height.
	VerifyNonMembership(ctx sdk.Context, clientID string, height ibcexported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path ibcexported.Path) error
	// UpdateClient updates the light client state with a new header
	UpdateClient(ctx sdk.Context, clientID string, clientMsg ibcexported.ClientMessage) error
	// GetClientStatus returns the status of a client given the client ID
	GetClientStatus(ctx sdk.Context, clientID string) ibcexported.Status
	// GetClientTimestampAtHeight returns the timestamp for a given height on the client
	// given its client ID and height
	GetClientTimestampAtHeight(ctx sdk.Context, clientID string, height ibcexported.Height) (uint64, error)
	// GetClientState gets a particular client from the store
	GetClientState(ctx sdk.Context, clientID string) (ibcexported.ClientState, bool)
	// Route returns the light client module for the given client identifier.
	Route(ctx sdk.Context, clientID string) (ibcexported.LightClientModule, error)
}
