package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// BankKeeper defines the expected bank keeper
type BankKeeper interface {
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	porttypes.ICS4Wrapper
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetChannelClientState(ctx sdk.Context, portID, channelID string) (clientID string, clientState exported.ClientState, err error)
	GetNextSequenceSend(ctx sdk.Context, sourcePort, sourceChannel string) (uint64, bool)
}

// ClientKeeper defines the expected IBC client keeper
type ClientKeeper interface {
	GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool)
	GetClientStatus(ctx sdk.Context, clientID string) exported.Status
}
