package keeper

import (
	"bytes"
	"context"
	"strings"

	wasm "github.com/CosmWasm/wasmvm"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/types"
)

var _ types.QueryServer = (*Keeper)(nil)
var _ types.MsgServer = (*Keeper)(nil)

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
	vm            *wasm.VM
}

func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, vmConfig *VMConfig, validationConfig *ValidationConfig) Keeper {
	supportedFeatures := strings.Join(vmConfig.SupportedFeatures, ",")

	vm, err := wasm.NewVM(vmConfig.DataDir, supportedFeatures, vmConfig.MemoryLimitMb, vmConfig.PrintDebug, vmConfig.CacheSizeMb)
	if err != nil {
		panic(err)
	}

	// This may potentially be unsafe, but we need to be able to create a VM for validation
	wasmValidator, err := NewWasmValidator(validationConfig, func() (*wasm.VM, error) {
		return wasm.NewVM(vmConfig.DataDir, supportedFeatures, vmConfig.MemoryLimitMb, vmConfig.PrintDebug, vmConfig.CacheSizeMb)
	})
	if err != nil {
		panic(err)
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      key,
		wasmValidator: wasmValidator,
		vm:            vm,
	}
}

func (k Keeper) SetWasmLightClient(ctx sdk.Context, code *types.WasmLightClient) error {
	store := ctx.KVStore(k.storeKey)

	// check to see if the store has a code with the same name
	if store.Has([]byte(code.Name)) {
		return types.ErrWasmCodeExists
	}

	// run the code through the wasm light client validation process
	if isValidWasmCode, err := k.wasmValidator.validateWasmCode(code.Code); err != nil {
		return sdkerrors.Wrapf(types.ErrWasmCodeValidation, "unable to validate wasm code: %s", err)
	} else if !isValidWasmCode {
		return types.ErrWasmInvalidCode
	}

	// create the code in the vm
	// TODO: do we need to check and make sure there
	// is no code with the same hash?
	codeID, err := k.vm.Create(code.Code)
	if err != nil {
		return types.ErrWasmInvalidCode
	}

	// safety check to assert that code id returned by WasmVM equals to code hash
	if !bytes.Equal(codeID, code.CodeHash) {
		return types.ErrWasmInvalidCodeID
	}

	// store the whole code in the store
	store.Set([]byte(code.Name), k.cdc.MustMarshalLengthPrefixed(code))
	return nil
}

func (k Keeper) GetWasmLightClients(ctx sdk.Context) []*types.WasmLightClient {
	// TODO: iterate over the store and return all wasm light clients
	return nil
}

func (k Keeper) GetWasmLightClient(ctx sdk.Context, name string) (out *types.WasmLightClient) {
	store := ctx.KVStore(k.storeKey)
	// TODO: this might be the wrong pattern but says what its supposed to do
	k.cdc.MustUnmarshalLengthPrefixed(store.Get([]byte(name)), out)
	return
}

func (k Keeper) SubmitWasmLightClient(ctx context.Context, in *types.MsgSubmitWasmLightClient) (*types.MsgSubmitWasmLightClientResponse, error) {
	if err := k.SetWasmLightClient(sdk.UnwrapSDKContext(ctx), in.WasmLightClient); err != nil {
		return nil, err
	}
	return &types.MsgSubmitWasmLightClientResponse{}, nil
}

func (k Keeper) WasmLightClient(ctx context.Context, in *types.WasmLightClientRequest) (*types.WasmLightClientResponse, error) {
	return &types.WasmLightClientResponse{WasmLightClient: k.GetWasmLightClient(sdk.UnwrapSDKContext(ctx), in.Name)}, nil
}
