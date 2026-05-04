package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/tokenfactory/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServer returns an implementation of the MsgServer interface for the provided Keeper.
func NewMsgServer(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreateDenom implements types.MsgServer. This function is used to create a new denom.
// And sets the msg.Sender as the admin of the denom.
func (k msgServer) CreateDenom(goCtx context.Context, msg *types.MsgCreateDenom) (*types.MsgCreateDenomResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "error: %s", err.Error())
	}

	if err := k.Keeper.CreateDenom(ctx, msg.Sender, msg.Denom); err != nil {
		return nil, err
	}

	return &types.MsgCreateDenomResponse{}, nil
}

// Mint implements types.MsgServer. This function is used to mint tokens to msg.MintToAddress.
// msg.Sender must be the admin of the denom.
func (k msgServer) Mint(ctx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	adminAddr, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "error: %s", err.Error())
	}

	mintToAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "error: %s", err.Error())
	}

	// authorization and denom existence checks are handled here (not in keeper)
	if err := k.validateMintBurnPermission(ctx, msg.From, msg.Amount.Denom); err != nil {
		return nil, err
	}

	if err := k.mintToWithAdmin(ctx, adminAddr, msg.Amount, mintToAddr); err != nil {
		return nil, err
	}

	return &types.MsgMintResponse{}, nil
}

// Burn implements types.MsgServer. This function is used to burn tokens from msg.BurnFromAddress.
// msg.Sender must be the admin of the denom.
func (k msgServer) Burn(ctx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	adminAddr, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "error: %s", err.Error())
	}

	// authorization and denom existence checks are handled here (not in keeper)
	if err := k.validateMintBurnPermission(ctx, msg.From, msg.Amount.Denom); err != nil {
		return nil, err
	}

	if err := k.burnFromWithAdmin(ctx, adminAddr, msg.Amount); err != nil {
		return nil, err
	}

	return &types.MsgBurnResponse{}, nil
}

// ChangeAdmin implements types.MsgServer. Transfers admin authority to a new address.
func (k msgServer) ChangeAdmin(ctx context.Context, msg *types.MsgChangeAdmin) (*types.MsgChangeAdminResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid sender: %s", err.Error())
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewAdmin); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid new admin: %s", err.Error())
	}

	if err := types.ValidateTokenFactoryDenom(msg.Denom); err != nil {
		return nil, err
	}

	if err := k.Keeper.ChangeAdmin(ctx, msg.Denom, msg.Sender, msg.NewAdmin); err != nil {
		return nil, err
	}

	return &types.MsgChangeAdminResponse{}, nil
}

// RenounceAdmin implements types.MsgServer. Permanently removes admin authority.
func (k msgServer) RenounceAdmin(ctx context.Context, msg *types.MsgRenounceAdmin) (*types.MsgRenounceAdminResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid sender: %s", err.Error())
	}

	if err := types.ValidateTokenFactoryDenom(msg.Denom); err != nil {
		return nil, err
	}

	if err := k.Keeper.RenounceAdmin(ctx, msg.Denom, msg.Sender); err != nil {
		return nil, err
	}

	return &types.MsgRenounceAdminResponse{}, nil
}
