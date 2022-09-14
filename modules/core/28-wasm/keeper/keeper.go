package keeper

import (
	"bytes"
	"crypto/sha256"
	"strings"

	wasm "github.com/CosmWasm/wasmvm"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/types"
)

// WasmVM initialized by wasm keeper
var WasmVM *wasm.VM

// VMConfig represents Wasm virtual machine settings
type VMConfig struct {
	DataDir           string
	SupportedFeatures []string
	MemoryLimitMb     uint32
	PrintDebug        bool
	CacheSizeMb       uint32
}

// Keeper will have a reference to Wasmer with it's own data directory.
type Keeper struct {
	storeKey      storetypes.StoreKey
	cdc           codec.BinaryCodec
	wasmValidator *WasmValidator
}

func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, vmConfig *VMConfig, validationConfig *ValidationConfig) Keeper {
	supportedFeatures := strings.Join(vmConfig.SupportedFeatures, ",")

	vm, err := wasm.NewVM(vmConfig.DataDir, supportedFeatures, vmConfig.MemoryLimitMb, vmConfig.PrintDebug, vmConfig.CacheSizeMb)
	if err != nil {
		panic(err)
	}

	wasmValidator, err := NewWasmValidator(validationConfig, func() (*wasm.VM, error) {
		return wasm.NewVM(vmConfig.DataDir, supportedFeatures, vmConfig.MemoryLimitMb, vmConfig.PrintDebug, vmConfig.CacheSizeMb)
	})
	if err != nil {
		panic(err)
	}

	WasmVM = vm

	return Keeper{
		cdc:           cdc,
		storeKey:      key,
		wasmValidator: wasmValidator,
	}
}

func (k Keeper) PushNewWasmCode(ctx sdk.Context, code []byte) ([]byte, error) {
	store := ctx.KVStore(k.storeKey)
	codeHash := generateWasmCodeHash(code)
	codeIDKey := types.CodeID(codeHash)

	if store.Has(codeIDKey) {
		return nil, types.ErrWasmCodeExists
	}

	if isValidWasmCode, err := k.wasmValidator.validateWasmCode(code); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrWasmCodeValidation, "unable to validate wasm code: %s", err)
	} else if !isValidWasmCode {
		return nil, types.ErrWasmInvalidCode
	}

	codeID, err := WasmVM.Create(code)
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
