package keeper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"

	cosmwasm "github.com/CosmWasm/wasmvm"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errorsmod "cosmossdk.io/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

// Keeper defines the 08-wasm keeper
type Keeper struct {
	// implements gRPC QueryServer interface
	types.QueryServer

	storeKey  storetypes.StoreKey
	cdc       codec.BinaryCodec
	wasmVM    *cosmwasm.VM
	authority string
}

// NewKeeper creates a new NewKeeper instance
func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey) Keeper {
	// Wasm VM
	const wasmDataDir = "ibc_08-wasm_client_data"
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

// GetAuthority returns the 08-wasm module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func generateWasmCodeHash(code []byte) []byte {
	hash := sha256.Sum256(code)
	return hash[:]
}

func (k Keeper) storeWasmCode(ctx sdk.Context, code []byte) ([]byte, error) {
	store := ctx.KVStore(k.storeKey)

	var err error
	if types.IsGzip(code) {
		ctx.GasMeter().ConsumeGas(types.VMGasRegister.UncompressCosts(len(code)), "Uncompress gzip bytecode")
		code, err = types.Uncompress(code, types.MaxWasmByteSize())
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to store contract")
		}
	}

	// Check to see if the store has a code with the same code it
	expectedHash := generateWasmCodeHash(code)
	codeIDKey := types.CodeIDKey(expectedHash)
	if store.Has(codeIDKey) {
		return nil, types.ErrWasmCodeExists
	}

	// run the code through the wasm light client validation process
	if err := types.ValidateWasmCode(code); err != nil {
		return nil, errorsmod.Wrapf(err, "wasm bytecode validation failed")
	}

	// create the code in the vm
	ctx.GasMeter().ConsumeGas(types.VMGasRegister.CompileCosts(len(code)), "Compiling wasm bytecode")
	codeHash, err := k.wasmVM.StoreCode(code)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to store contract")
	}

	// safety check to assert that code ID returned by WasmVM equals to code hash
	if !bytes.Equal(codeHash, expectedHash) {
		return nil, errorsmod.Wrapf(types.ErrInvalidCodeID, "expected %s, got %s", hex.EncodeToString(expectedHash), hex.EncodeToString(codeHash))
	}

	store.Set(codeIDKey, code)
	return codeHash, nil
}

func (k Keeper) importWasmCode(ctx sdk.Context, codeIDKey, wasmCode []byte) error {
	store := ctx.KVStore(k.storeKey)
	if types.IsGzip(wasmCode) {
		var err error
		wasmCode, err = types.Uncompress(wasmCode, types.MaxWasmByteSize())
		if err != nil {
			return errorsmod.Wrap(err, "failed to store contract")
		}
	}

	generatedCodeID, err := k.wasmVM.Create(wasmCode)
	if err != nil {
		return errorsmod.Wrap(err, "failed to store contract")
	}
	generatedCodeIDKey := types.CodeIDKey(generatedCodeID)

	if !bytes.Equal(codeIDKey, generatedCodeIDKey) {
		return errorsmod.Wrapf(types.ErrInvalid, "expected %s, got %s", string(generatedCodeIDKey), string(codeIDKey))
	}

	store.Set(codeIDKey, wasmCode)
	return nil
}
