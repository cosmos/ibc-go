package keeper

import (
	"context"
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v11/types"
)

var _ types.MsgServer = (*Keeper)(nil)

// StoreCode defines a rpc handler method for MsgStoreCode
func (k *Keeper) StoreCode(goCtx context.Context, msg *types.MsgStoreCode) (*types.MsgStoreCodeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := ctx.ValidateAuthority(k.GetAuthority(), msg.Signer); err != nil {
		return nil, err
	}
	checksum, err := k.storeWasmCode(ctx, msg.WasmByteCode, k.GetVM().StoreCode)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to store wasm bytecode")
	}

	emitStoreWasmCodeEvent(ctx, checksum)

	return &types.MsgStoreCodeResponse{
		Checksum: checksum,
	}, nil
}

// RemoveChecksum defines a rpc handler method for MsgRemoveChecksum
func (k *Keeper) RemoveChecksum(goCtx context.Context, msg *types.MsgRemoveChecksum) (*types.MsgRemoveChecksumResponse,
	error,
) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := ctx.ValidateAuthority(k.GetAuthority(), msg.Signer); err != nil {
		return nil, err
	}

	if !k.HasChecksum(ctx, msg.Checksum) {
		return nil, types.ErrWasmChecksumNotFound
	}

	err := k.GetChecksums().Remove(goCtx, msg.Checksum)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to remove checksum")
	}

	// unpin the code from the vm in-memory cache
	if err := k.GetVM().Unpin(msg.Checksum); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to unpin contract with checksum (%s) from vm cache", hex.EncodeToString(msg.Checksum))
	}

	return &types.MsgRemoveChecksumResponse{}, nil
}

// MigrateContract defines a rpc handler method for MsgMigrateContract
func (k *Keeper) MigrateContract(goCtx context.Context, msg *types.MsgMigrateContract) (*types.MsgMigrateContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := ctx.ValidateAuthority(k.GetAuthority(), msg.Signer); err != nil {
		return nil, err
	}

	err := k.migrateContractCode(ctx, msg.ClientId, msg.Checksum, msg.Msg)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to migrate contract")
	}

	// event emission is handled in migrateContractCode

	return &types.MsgMigrateContractResponse{}, nil
}
