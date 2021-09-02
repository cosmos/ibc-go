package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/tendermint/tendermint/libs/bytes"
)

// BankKeeper defines the expected bank keeper
type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
}

// TransferKeeper defines the expected transfer keeper
type TransferKeeper interface {
	SendTransfer(ctx sdk.Context, sourcePort, sourceChannel string, token sdk.Coin, sender sdk.AccAddress, receiver string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64) error
	GetReceiveEnabled(ctx sdk.Context) bool
	GetSendEnabled(ctx sdk.Context) bool
	HasDenomTrace(ctx sdk.Context, traceHash bytes.HexBytes) bool
	SetDenomTrace(ctx sdk.Context, traceHash bytes.HexBytes)
}
