package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	clienttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

// AccountKeeper defines the contract required for account APIs.
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, name string) sdk.ModuleAccountI
}

// BankKeeper defines the expected bank keeper
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	BlockedAddr(addr sdk.AccAddress) bool
	IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error
	HasDenomMetaData(ctx context.Context, denom string) bool
	SetDenomMetaData(ctx context.Context, denomMetaData banktypes.Metadata)
	SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
	GetAllChannelsWithPortPrefix(ctx sdk.Context, portPrefix string) []channeltypes.IdentifiedChannel
	HasChannel(ctx sdk.Context, portID, channelID string) bool
}

// MessageRouter ADR 031 request type routing
// https://github.com/cosmos/cosmos-sdk/blob/main/docs/architecture/adr-031-msg-service.md
type MessageRouter interface {
	Handler(msg sdk.Msg) baseapp.MsgServiceHandler
}

// ChannelKeeperV2 defines the expected IBC channelV2 keeper
type ChannelKeeperV2 interface {
	SendPacket(ctx context.Context, msg *channeltypesv2.MsgSendPacket) (*channeltypesv2.MsgSendPacketResponse, error)
}

// ClientKeeper defines the expected IBC client keeper
type ClientKeeperV2 interface {
	GetClientCounterparty(ctx sdk.Context, clientID string) (counterparty clienttypesv2.CounterpartyInfo, found bool)
}

// ConnectionKeeper defines the expected IBC connection keeper
type ConnectionKeeper interface {
	GetConnection(ctx sdk.Context, connectionID string) (connection connectiontypes.ConnectionEnd, found bool)
}

// ParamSubspace defines the expected Subspace interface for module parameters.
type ParamSubspace interface {
	GetParamSet(ctx sdk.Context, ps paramtypes.ParamSet)
}
