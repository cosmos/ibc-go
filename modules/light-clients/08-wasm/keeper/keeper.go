package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cosmwasm "github.com/CosmWasm/wasmvm"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

type Keeper struct {
	storeKey  storetypes.StoreKey
	cdc       codec.BinaryCodec
	wasmVM    *cosmwasm.VM
	authority string
}

func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey) Keeper {
	// Wasm VM
	wasmDataDir := "wasm_client_data"
	wasmSupportedFeatures := strings.Join([]string{"storage", "iterator"}, ",")
	wasmMemoryLimitMb := uint32(math.Pow(2, 12))
	wasmPrintDebug := true
	wasmCacheSizeMb := uint32(math.Pow(2, 8))

	vm, err := cosmwasm.NewVM(wasmDataDir, wasmSupportedFeatures, wasmMemoryLimitMb, wasmPrintDebug, wasmCacheSizeMb)
	if err != nil {
		panic(err)
	}
	types.WasmVM = vm

	// governance authority
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	return Keeper{
		cdc:       cdc,
		storeKey:  key,
		wasmVM:    vm,
		authority: authority.String(),
	}
}

func (k Keeper) storeWasmCode(ctx sdk.Context, code []byte) ([]byte, error) {
	store := ctx.KVStore(k.storeKey)

	// Check to see if the store has a code with the same code it
	codeHash := generateWasmCodeHash(code)
	codeIDKey := types.CodeID(codeHash)
	if store.Has(codeIDKey) {
		return nil, types.ErrWasmCodeExists
	}

	// run the code through the wasm light client validation process
	if isValidWasmCode, err := types.ValidateWasmCode(code); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrWasmCodeValidation, "unable to validate wasm code: %s", err)
	} else if !isValidWasmCode {
		return nil, types.ErrWasmInvalidCode
	}

	// create the code in the vm
	codeID, err := types.WasmVM.Create(code)
	if err != nil {
		return nil, types.ErrWasmInvalidCode
	}

	// safety check to assert that code id returned by WasmVM equals to code hash
	if !bytes.Equal(codeID, codeHash) {
		return nil, types.ErrWasmInvalidCodeID
	}

	store.Set(codeIDKey, code)
	return codeID, nil
}

func generateWasmCodeHash(code []byte) []byte {
	hash := sha256.Sum256(code)
	return hash[:]
}

func (k Keeper) getWasmCode(c context.Context, query *types.WasmCodeQuery) (*types.WasmCodeResponse, error) {
	if query == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)

	codeID, err := hex.DecodeString(query.CodeId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid code id")
	}

	codeKey := types.CodeID(codeID)
	code := store.Get(codeKey)
	if code == nil {
		return nil, status.Error(
			codes.NotFound,
			sdkerrors.Wrap(types.ErrWasmCodeIDNotFound, query.CodeId).Error(),
		)
	}

	return &types.WasmCodeResponse{
		Code: code,
	}, nil
}
