package v7

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v6/modules/core/03-connection/types"
)

// ConnectionKeeper expected IBC connection keeper
type ConnectionKeeper interface {
	CreateSentinelLocalhostConnection() types.ConnectionEnd
	SetConnection(ctx sdk.Context, connectionID string, connection types.ConnectionEnd)
}
