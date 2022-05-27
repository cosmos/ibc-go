package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NFTKeeper defines the expected nft keeper
type NFTKeeper interface {
	SaveClass(ctx sdk.Context, classID, classURI string) error
	Mint(ctx sdk.Context, classID, tokenID, tokenURI, receiver string)
	Transfer(ctx sdk.Context, classID, tokenID, receiver string)
	Burn(ctx sdk.Context, classID, tokenID string)

	GetOwner(ctx sdk.Context, classID string, nftID string) sdk.AccAddress
	HasClass(ctx sdk.Context, classID string) bool
}
