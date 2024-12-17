package types

import (
	"context"

	paramtypes "cosmossdk.io/x/params/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// AuthKeeper defines the expected x/auth keeper
type AuthKeeper interface {
	NewAccount(ctx context.Context, acc sdk.AccountI) sdk.AccountI
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
	GetModuleAccount(ctx context.Context, name string) sdk.ModuleAccountI
	GetModuleAddress(name string) sdk.AccAddress
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(ctx context.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetNextSequenceSend(ctx context.Context, portID, channelID string) (uint64, bool)
	GetConnection(ctx context.Context, connectionID string) (connectiontypes.ConnectionEnd, error)
	GetAllChannelsWithPortPrefix(ctx context.Context, portPrefix string) []channeltypes.IdentifiedChannel
}

// ParamSubspace defines the expected Subspace interface for module parameters.
type ParamSubspace interface {
	GetParamSet(ctx sdk.Context, ps paramtypes.ParamSet)
	GetParamSetIfExists(ctx sdk.Context, ps paramtypes.ParamSet)
}
