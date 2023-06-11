package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// AccountKeeper defines the expected account keeper
type AccountKeeper interface {
	NewAccount(ctx sdk.Context, acc authtypes.AccountI) types.AccountI
	GetAccount(ctx context.Context, addr sdk.AccAddress) types.AccountI
	SetAccount(ctx sdk.Context, acc authtypes.AccountI)
	GetModuleAccount(ctx context.Context, name string) types.ModuleAccountI
	GetModuleAddress(name string) sdk.AccAddress
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
	GetConnection(ctx sdk.Context, connectionID string) (ibcexported.ConnectionI, error)
	GetAllChannelsWithPortPrefix(ctx sdk.Context, portPrefix string) []channeltypes.IdentifiedChannel
}

// PortKeeper defines the expected IBC port keeper
type PortKeeper interface {
	BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability
	IsBound(ctx sdk.Context, portID string) bool
}
