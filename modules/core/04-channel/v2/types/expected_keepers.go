package types

import (
	"context"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

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
