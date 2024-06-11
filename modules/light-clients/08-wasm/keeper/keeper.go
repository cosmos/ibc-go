package keeper

import (
	"bytes"
	"encoding/hex"

	wasmvm "github.com/CosmWasm/wasmvm/v2"

	storetypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
)

// Keeper defines the 08-wasm keeper
type Keeper struct {
	// implements gRPC QueryServer interface
	types.QueryServer

	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService

	clientKeeper types.ClientKeeper

	authority string
}

// Codec returns the 08-wasm module's codec.
func (k Keeper) Codec() codec.BinaryCodec {
	return k.cdc
}

// GetAuthority returns the 08-wasm module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (Keeper) storeWasmCode(ctx sdk.Context, code []byte, storeFn func(code wasmvm.WasmCode, gasLimit uint64) (wasmvm.Checksum, uint64, error)) ([]byte, error) {
	var err error
	if types.IsGzip(code) {
		ctx.GasMeter().ConsumeGas(types.VMGasRegister.UncompressCosts(len(code)), "Uncompress gzip bytecode")
		code, err = types.Uncompress(code, types.MaxWasmByteSize())
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to store contract")
		}
	}

	// run the code through the wasm light client validation process
	if err := types.ValidateWasmCode(code); err != nil {
		return nil, errorsmod.Wrap(err, "wasm bytecode validation failed")
	}

	// Check to see if store already has checksum.
	checksum, err := types.CreateChecksum(code)
	if err != nil {
		return nil, errorsmod.Wrap(err, "wasm bytecode checksum failed")
	}

	if types.HasChecksum(ctx, checksum) {
		return nil, types.ErrWasmCodeExists
	}

	// create the code in the vm
	gasLeft := types.VMGasRegister.RuntimeGasForContract(ctx)
	vmChecksum, gasUsed, err := storeFn(code, gasLeft)
	types.VMGasRegister.ConsumeRuntimeGas(ctx, gasUsed)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to store contract")
	}

	// SANITY: We've checked our store, additional safety check to assert that the checksum returned by WasmVM equals checksum generated by us.
	if !bytes.Equal(vmChecksum, checksum) {
		return nil, errorsmod.Wrapf(types.ErrInvalidChecksum, "expected %s, got %s", hex.EncodeToString(checksum), hex.EncodeToString(vmChecksum))
	}

	// pin the code to the vm in-memory cache
	if err := ibcwasm.GetVM().Pin(vmChecksum); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to pin contract with checksum (%s) to vm cache", hex.EncodeToString(vmChecksum))
	}

	// store the checksum
	err = ibcwasm.Checksums.Set(ctx, checksum)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to store checksum")
	}

	return checksum, nil
}

func (k Keeper) migrateContractCode(ctx sdk.Context, clientID string, newChecksum, migrateMsg []byte) error {
	wasmClientState, err := k.GetWasmClientState(ctx, clientID)
	if err != nil {
		return errorsmod.Wrap(err, "failed to retrieve wasm client state")
	}
	oldChecksum := wasmClientState.Checksum

	clientStore := k.clientKeeper.ClientStore(ctx, clientID)

	err = wasmClientState.MigrateContract(ctx, k.cdc, clientStore, clientID, newChecksum, migrateMsg)
	if err != nil {
		return errorsmod.Wrap(err, "contract migration failed")
	}

	// client state may be updated by the contract migration
	wasmClientState, err = k.GetWasmClientState(ctx, clientID)
	if err != nil {
		// note that this also ensures that the updated client state is
		// still a wasm client state
		return errorsmod.Wrap(err, "failed to retrieve the updated wasm client state")
	}

	// update the client state checksum before persisting it
	wasmClientState.Checksum = newChecksum

	k.clientKeeper.SetClientState(ctx, clientID, wasmClientState)

	emitMigrateContractEvent(ctx, clientID, oldChecksum, newChecksum)

	return nil
}

// GetWasmClientState returns the 08-wasm client state for the given client identifier.
func (k Keeper) GetWasmClientState(ctx sdk.Context, clientID string) (*types.ClientState, error) {
	clientState, found := k.clientKeeper.GetClientState(ctx, clientID)
	if !found {
		return nil, errorsmod.Wrapf(clienttypes.ErrClientTypeNotFound, "clientID %s", clientID)
	}

	wasmClientState, ok := clientState.(*types.ClientState)
	if !ok {
		return nil, errorsmod.Wrapf(clienttypes.ErrInvalidClient, "expected type %T, got %T", (*types.ClientState)(nil), wasmClientState)
	}

	return wasmClientState, nil
}

// InitializePinnedCodes updates wasmvm to pin to cache all contracts marked as pinned
func InitializePinnedCodes(ctx sdk.Context) error {
	checksums, err := types.GetAllChecksums(ctx)
	if err != nil {
		return err
	}

	for _, checksum := range checksums {
		if err := ibcwasm.GetVM().Pin(checksum); err != nil {
			return err
		}
	}
	return nil
}
