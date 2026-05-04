package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/tokenfactory/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MintTo mints tokens of denom to address.
// MUST fail if denom is not recognized or not authorized for minting.
func (k Keeper) MintTo(ctx context.Context, denom string, amount math.Int, to sdk.AccAddress) error {
	if err := types.ValidateTokenFactoryDenom(denom); err != nil {
		return err
	}

	coin := sdk.NewCoin(denom, amount)
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return err
	}

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, to, sdk.NewCoins(coin)); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeEvtMint,
			sdk.NewAttribute(types.AttributeKeyDenom, denom),
			sdk.NewAttribute(types.AttributeKeyAmount, amount.String()),
			sdk.NewAttribute(types.AttributeKeyMintTo, to.String()),
		),
	)

	return nil
}

// BurnFrom burns tokens of denom from address.
// MUST fail if address does not have enough balance or burn is not permitted.
func (k Keeper) BurnFrom(ctx context.Context, denom string, amount math.Int, from sdk.AccAddress) error {
	if err := types.ValidateTokenFactoryDenom(denom); err != nil {
		return err
	}

	coin := sdk.NewCoin(denom, amount)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, from, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return err
	}

	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeEvtBurn,
			sdk.NewAttribute(types.AttributeKeyDenom, denom),
			sdk.NewAttribute(types.AttributeKeyAmount, amount.String()),
			sdk.NewAttribute(types.AttributeKeyBurnFrom, from.String()),
		),
	)

	return nil
}

// HasDenom checks if a denom exists in the token factory.
func (k Keeper) HasDenom(ctx context.Context, denom string) bool {
	if err := types.ValidateTokenFactoryDenom(denom); err != nil {
		return false
	}
	has, err := k.DenomAuthorityMetadataStore.Has(ctx, denom)
	if err != nil {
		return false
	}
	return has
}

// mintToWithAdmin mints coins to the specified address (admin-based, for MsgMint).
func (k Keeper) mintToWithAdmin(ctx context.Context, adminAddr sdk.AccAddress, amount sdk.Coin, mintToAddr sdk.AccAddress) error {
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(amount)); err != nil {
		return err
	}

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, mintToAddr, sdk.NewCoins(amount)); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeEvtMint,
			sdk.NewAttribute(types.AttributeKeyDenom, amount.Denom),
			sdk.NewAttribute(types.AttributeKeyAmount, amount.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyAdmin, adminAddr.String()),
			sdk.NewAttribute(types.AttributeKeyMintTo, mintToAddr.String()),
		),
	)

	return nil
}

// burnFromWithAdmin burns coins from the specified address (admin-based, for MsgBurn).
func (k Keeper) burnFromWithAdmin(ctx context.Context, from sdk.AccAddress, amount sdk.Coin) error {
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, from, types.ModuleName, sdk.NewCoins(amount)); err != nil {
		return err
	}

	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(amount)); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeEvtBurn,
			sdk.NewAttribute(types.AttributeKeyDenom, amount.Denom),
			sdk.NewAttribute(types.AttributeKeyAmount, amount.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyAdmin, from.String()),
		),
	)

	return nil
}

func (k Keeper) validateMintBurnPermission(ctx context.Context, admin, denom string) error {
	if err := types.ValidateTokenFactoryDenom(denom); err != nil {
		return err
	}

	md, err := k.GetAuthorityMetadata(ctx, denom)
	if err != nil {
		return errorsmod.Wrapf(types.ErrDenomNotFound, "denom %s not found", denom)
	}
	if md.Admin == "" {
		return errorsmod.Wrapf(types.ErrAdminRenounced, "admin renounced for denom %s", denom)
	}
	if md.Admin != admin {
		return errorsmod.Wrapf(types.ErrUnauthorized, "admin %s not authorized for denom %s", admin, denom)
	}
	return nil
}
