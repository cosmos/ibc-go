package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// AuthKeeper defines the contract required for the x/auth keeper.
type AuthKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(ctx context.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetPacketCommitment(ctx context.Context, portID, channelID string, sequence uint64) []byte
	GetNextSequenceSend(ctx context.Context, portID, channelID string) (uint64, bool)
	HasChannel(ctx context.Context, portID, channelID string) bool
}

// BankKeeper defines the expected bank keeper
type BankKeeper interface {
	HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	BlockedAddr(sdk.AccAddress) bool
	IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error
}
