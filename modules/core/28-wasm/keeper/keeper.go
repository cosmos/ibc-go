package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	wasm "github.com/CosmWasm/wasmvm"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/28-wasm/types"
)

// WasmVM initialized by wasm keeper
var WasmVM *wasm.VM

// VMConfig represents WASM virtual machine settings
type VMConfig struct {
	DataDir           string
	SupportedFeatures []string
	MemoryLimitMb     uint32
	PrintDebug        bool
	CacheSizeMb       uint32
}

func generateWASMCodeHash(code []byte) string {
	hash := sha256.Sum256(code)
	return hex.EncodeToString(hash[:])
}

// Keeper will have a reference to Wasmer with it's own data directory.
type Keeper struct {
	storeKey      sdk.StoreKey
	cdc           codec.BinaryCodec
	wasmValidator *WASMValidator
}

func NewKeeper(cdc codec.BinaryCodec, key sdk.StoreKey, vmConfig *VMConfig, validationConfig *ValidationConfig) Keeper {
	supportedFeatures := strings.Join(vmConfig.SupportedFeatures, ",")

	vm, err := wasm.NewVM(vmConfig.DataDir, supportedFeatures, vmConfig.MemoryLimitMb, vmConfig.PrintDebug, vmConfig.CacheSizeMb)
	if err != nil {
		panic(err)
	}

	wasmValidator, err := NewWASMValidator(validationConfig, func() (*wasm.VM, error) {
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

func (k Keeper) PushNewWASMCode(ctx sdk.Context, clientType string, code []byte) ([]byte, string, error) {
	store := ctx.KVStore(k.storeKey)
	codeHash := generateWASMCodeHash(code)

	latestVersionKey := host.LatestWASMCode(clientType)

	if isValidWASMCode, err := k.wasmValidator.validateWASMCode(code); err != nil {
		return nil, "", fmt.Errorf("unable to validate wasm code, error: %s", err)
	} else if !isValidWASMCode {
		return nil, "", fmt.Errorf("invalid wasm code")
	}

	codeID, err := WasmVM.Create(code)
	if err != nil {
		return nil, "", fmt.Errorf("invalid wasm code")
	}

	codekey := host.WASMCode(clientType, codeHash)
	entryKey := host.WASMCodeEntry(clientType, codeHash)

	latestVersionCodeHash := store.Get(latestVersionKey)

	// More careful management of doubly linked list can lift this constraint
	// But we do not see any significant advantage of it.
	if store.Has(entryKey) {
		return nil, "", fmt.Errorf("wasm code already exists")
	}

	codeEntry := types.WasmCodeEntry{
		PreviousCodeHash: string(latestVersionCodeHash),
		NextCodeHash:     "",
		CodeId:           codeID,
	}

	previousVersionEntryKey := host.WASMCodeEntry(clientType, string(latestVersionCodeHash))
	previousVersionEntryBz := store.Get(previousVersionEntryKey)
	if len(previousVersionEntryBz) != 0 {
		var previousEntry types.WasmCodeEntry
		k.cdc.MustUnmarshal(previousVersionEntryBz, &previousEntry)
		previousEntry.NextCodeHash = codeHash
		store.Set(previousVersionEntryKey, k.cdc.MustMarshal(&previousEntry))
	}

	store.Set(entryKey, k.cdc.MustMarshal(&codeEntry))
	store.Set(latestVersionKey, []byte(codeHash))
	store.Set(codekey, code)

	return codeID, codeHash, nil
}
