package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
)

// TransferKeeper defines the expected transfer keeper
type TransferKeeper interface {
	Transfer(ctx context.Context, msg *transfertypes.MsgTransfer) (*transfertypes.MsgTransferResponse, error)
	GetDenom(ctx sdk.Context, denomHash cmtbytes.HexBytes) (transfertypes.Denom, bool)
	GetTotalEscrowForDenom(ctx sdk.Context, denom string) sdk.Coin
	SetTotalEscrowForDenom(ctx sdk.Context, coin sdk.Coin)
	DenomPathFromHash(ctx sdk.Context, ibcDenom string) (string, error)

	// Only used for v3 migration
	GetPort(ctx sdk.Context) string
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	porttypes.ICS4Wrapper
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)

	// Only used for v3 migration
	GetAllChannelsWithPortPrefix(ctx sdk.Context, portPrefix string) []channeltypes.IdentifiedChannel
}

// BankKeeper defines the expected bank keeper
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error

	// Only used for v3 migration
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}
