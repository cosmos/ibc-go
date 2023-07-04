package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"path/filepath"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cosmwasm "github.com/CosmWasm/wasmvm"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

type Keeper struct {
	storeKey  storetypes.StoreKey
	cdc       codec.BinaryCodec
	wasmVM    *cosmwasm.VM
	authority string
}

func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, authority string, homeDir string) Keeper {
	// Wasm VM
	wasmDataDir := filepath.Join(homeDir, "wasm_client_data")
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

	return Keeper{
		cdc:       cdc,
		storeKey:  key,
		wasmVM:    vm,
		authority: authority,
	}
}

func (k Keeper) storeWasmCode(ctx sdk.Context, code []byte) ([]byte, error) {
	store := ctx.KVStore(k.storeKey)

	var err error
	if IsGzip(code) {
		ctx.GasMeter().ConsumeGas(types.VMGasRegister.UncompressCosts(len(code)), "Uncompress gzip bytecode")
		code, err = Uncompress(code, uint64(types.MaxWasmSize))
		if err != nil {
			return nil, sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
		}
	}

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
	ctx.GasMeter().ConsumeGas(types.VMGasRegister.CompileCosts(len(code)), "Compiling wasm bytecode")
	codeID, err := types.WasmVM.Create(code)
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrWasmInvalidCode, "unable to compile wasm code: %s", err)
	}

	// safety check to assert that code id returned by WasmVM equals to code hash
	if !bytes.Equal(codeID, codeHash) {
		return nil, types.ErrWasmInvalidCodeID
	}

	store.Set(codeIDKey, code)
	return codeID, nil
}

func (k Keeper) importWasmCode(ctx sdk.Context, codeHash, wasmCode []byte) error {
	store := ctx.KVStore(k.storeKey)
	if IsGzip(wasmCode) {
		var err error
		wasmCode, err = Uncompress(wasmCode, uint64(types.MaxWasmSize))
		if err != nil {
			return sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
		}
	}
	newCodeHash, err := k.wasmVM.Create(wasmCode)
	if err != nil {
		return sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	if !bytes.Equal(codeHash, types.CodeID(newCodeHash)) {
		return sdkerrors.Wrap(types.ErrInvalid, "code hashes not same")
	}

	store.Set(codeHash, wasmCode)
	return nil
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

func (k Keeper) getAllWasmCodeID(c context.Context, query *types.AllWasmCodeIDQuery) (*types.AllWasmCodeIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	var allCode []string

	store := ctx.KVStore(k.storeKey)
	prefixStore := prefix.NewStore(store, types.PrefixCodeIDKey)

	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()

	pageRes, err := sdkquery.FilteredPaginate(prefixStore, query.Pagination, func(key []byte, _ []byte, accumulate bool) (bool, error) {
		if accumulate {
			allCode = append(allCode, string(key))
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.AllWasmCodeIDResponse{
		CodeIds:    allCode,
		Pagination: pageRes,
	}, nil
}

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) error {
	for _, contract := range gs.Contracts {
		err := k.importWasmCode(ctx, contract.CodeHash, contract.ContractCode)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.PrefixCodeIDKey)
	defer iterator.Close()

	var genesisState types.GenesisState
	for ; iterator.Valid(); iterator.Next() {
		genesisState.Contracts = append(genesisState.Contracts, types.GenesisContract{
			CodeHash:     iterator.Key(),
			ContractCode: iterator.Value(),
		})
	}
	return genesisState
}
