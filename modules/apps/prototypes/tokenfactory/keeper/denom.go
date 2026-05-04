package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/tokenfactory/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// CreateDenom creates a new tokenfactory denom
func (k Keeper) CreateDenom(ctx context.Context, creatorAddr string, denom string) error {
	if _, found := k.bankKeeper.GetDenomMetaData(ctx, denom); found {
		return errorsmod.Wrapf(types.ErrDenomExists, "denom: %s", denom)
	}

	denomMetaData := banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{{
			Denom:    denom,
			Exponent: 0,
		}},
		Base: denom,
	}

	k.bankKeeper.SetDenomMetaData(ctx, denomMetaData)

	authorityMetadata := types.DenomAuthorityMetadata{
		Admin: creatorAddr,
	}

	if err := k.setAuthorityMetadata(ctx, denom, authorityMetadata); err != nil {
		return err
	}

	if err := k.addDenomFromCreator(ctx, creatorAddr, denom); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeEvtCreateDenom,
			sdk.NewAttribute(types.AttributeKeyDenom, denom),
			sdk.NewAttribute(types.AttributeKeyAdmin, creatorAddr),
		),
	)

	return nil
}

// GetAuthorityMetadata returns the authority metadata for a specific denom
func (k Keeper) GetAuthorityMetadata(ctx context.Context, denom string) (types.DenomAuthorityMetadata, error) {
	return k.DenomAuthorityMetadataStore.Get(ctx, denom)
}

// GetDenomsFromCreator returns all denoms created by a specific creator
func (k Keeper) GetDenomsFromCreator(ctx context.Context, creator string) ([]string, error) {
	denoms := []string{}

	err := k.CreatorPrefixStore.Walk(ctx, collections.NewPrefixedPairRange[string, string](creator), func(key collections.Pair[string, string], value bool) (bool, error) {
		denoms = append(denoms, key.K2())
		return false, nil
	})

	return denoms, err
}

func (k Keeper) setAuthorityMetadata(ctx context.Context, denom string, metadata types.DenomAuthorityMetadata) error {
	return k.DenomAuthorityMetadataStore.Set(ctx, denom, metadata)
}

func (k Keeper) addDenomFromCreator(ctx context.Context, creator, denom string) error {
	return k.CreatorPrefixStore.Set(ctx, collections.Join(creator, denom), true)
}

// HasDenomAuthorityMetadata checks if authority metadata exists for a denom
func (k Keeper) HasDenomAuthorityMetadata(ctx context.Context, denom string) bool {
	has, err := k.DenomAuthorityMetadataStore.Has(ctx, denom)
	if err != nil {
		return false
	}
	return has
}

// ChangeAdmin transfers admin authority to a new address
func (k Keeper) ChangeAdmin(ctx context.Context, denom, currentAdmin, newAdmin string) error {
	md, err := k.GetAuthorityMetadata(ctx, denom)
	if err != nil {
		return errorsmod.Wrapf(types.ErrDenomNotFound, "denom %s not found", denom)
	}

	if md.Admin == "" {
		return errorsmod.Wrapf(types.ErrAdminRenounced, "cannot change admin for denom %s", denom)
	}

	if md.Admin != currentAdmin {
		return errorsmod.Wrapf(types.ErrUnauthorized, "sender %s is not admin of denom %s", currentAdmin, denom)
	}

	md.Admin = newAdmin
	if err := k.setAuthorityMetadata(ctx, denom, md); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeEvtChangeAdmin,
			sdk.NewAttribute(types.AttributeKeyDenom, denom),
			sdk.NewAttribute(types.AttributeKeyAdmin, currentAdmin),
			sdk.NewAttribute(types.AttributeKeyNewAdmin, newAdmin),
		),
	)

	return nil
}

// RenounceAdmin permanently removes admin authority
func (k Keeper) RenounceAdmin(ctx context.Context, denom, currentAdmin string) error {
	md, err := k.GetAuthorityMetadata(ctx, denom)
	if err != nil {
		return errorsmod.Wrapf(types.ErrDenomNotFound, "denom %s not found", denom)
	}

	if md.Admin == "" {
		return errorsmod.Wrapf(types.ErrAdminRenounced, "admin already renounced for denom %s", denom)
	}

	if md.Admin != currentAdmin {
		return errorsmod.Wrapf(types.ErrUnauthorized, "sender %s is not admin of denom %s", currentAdmin, denom)
	}

	md.Admin = ""
	if err := k.setAuthorityMetadata(ctx, denom, md); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeEvtRenounceAdmin,
			sdk.NewAttribute(types.AttributeKeyDenom, denom),
			sdk.NewAttribute(types.AttributeKeyAdmin, currentAdmin),
		),
	)

	return nil
}
