package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// AccountKeeper defines the expected account keeper
type AccountKeeper interface {
	NewAccount(ctx sdk.Context, acc sdk.AccountI) sdk.AccountI
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx sdk.Context, acc sdk.AccountI)
	GetModuleAccount(ctx sdk.Context, name string) sdk.ModuleAccountI
	GetModuleAddress(name string) sdk.AccAddress
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
	GetConnection(ctx sdk.Context, connectionID string) (connectiontypes.ConnectionEnd, error)
	GetAllChannelsWithPortPrefix(ctx sdk.Context, portPrefix string) []channeltypes.IdentifiedChannel
}

// ParamSubspace defines the expected Subspace interface for module parameters.
type ParamSubspace interface {
	GetParamSet(ctx sdk.Context, ps paramtypes.ParamSet)
	GetParamSetIfExists(ctx sdk.Context, ps paramtypes.ParamSet)
}
