package keeper

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

var (
	VMGasRegister = types.NewDefaultWasmGasRegister()
	// wasmvmAPI is a wasmvm.GoAPI implementation that is passed to the wasmvm, it
	// doesn't implement any functionality, directly returning an error.
	wasmvmAPI = wasmvm.GoAPI{
		HumanizeAddress:     humanizeAddress,
		CanonicalizeAddress: canonicalizeAddress,
		ValidateAddress:     validateAddress,
	}
)

// WasmQuery queries the contract with the given payload and returns the result.
// WasmQuery returns an error if:
// - the payload cannot be marshaled to JSON
// - the contract query returns an error
// - the data bytes of the response cannot be unmarshal into the result type
func WasmQuery[T types.ContractResult](ctx sdk.Context, k Keeper, vm ibcwasm.WasmEngine, clientID string, clientStore storetypes.KVStore, cs *types.ClientState, payload types.QueryMsg) (T, error) {
	var result T

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return result, errorsmod.Wrap(err, "failed to marshal payload for wasm query")
	}

	res, err := k.queryContract(ctx, vm, clientID, clientStore, cs.Checksum, encodedData)
	if err != nil {
		return result, errorsmod.Wrap(types.ErrVMError, err.Error())
	}
	if res.Err != "" {
		return result, errorsmod.Wrap(types.ErrWasmContractCallFailed, res.Err)
	}

	if err := json.Unmarshal(res.Ok, &result); err != nil {
		return result, errorsmod.Wrapf(types.ErrWasmInvalidResponseData, "failed to unmarshal result of wasm query: %v", err)
	}

	return result, nil
}

// WasmSudo calls the contract with the given payload and returns the result.
// WasmSudo returns an error if:
// - the payload cannot be marshaled to JSON
// - the contract call returns an error
// - the response of the contract call contains non-empty messages
// - the response of the contract call contains non-empty events
// - the response of the contract call contains non-empty attributes
// - the data bytes of the response cannot be unmarshaled into the result type
func WasmSudo[T types.ContractResult](ctx sdk.Context, k Keeper, vm ibcwasm.WasmEngine, clientID string, cdc codec.BinaryCodec, clientStore storetypes.KVStore, cs *types.ClientState, payload types.SudoMsg) (T, error) {
	var result T

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return result, errorsmod.Wrap(err, "failed to marshal payload for wasm execution")
	}

	checksum := cs.Checksum
	res, err := k.callContract(ctx, vm, clientID, clientStore, checksum, encodedData)
	if err != nil {
		return result, errorsmod.Wrap(types.ErrVMError, err.Error())
	}
	if res.Err != "" {
		return result, errorsmod.Wrap(types.ErrWasmContractCallFailed, res.Err)
	}

	if err = types.CheckResponse(res.Ok); err != nil {
		return result, errorsmod.Wrapf(err, "checksum (%s)", hex.EncodeToString(cs.Checksum))
	}

	if err := json.Unmarshal(res.Ok.Data, &result); err != nil {
		return result, errorsmod.Wrap(types.ErrWasmInvalidResponseData, err.Error())
	}

	newClientState, err := types.ValidatePostExecutionClientState(clientStore, cdc)
	if err != nil {
		return result, err
	}

	// Checksum should only be able to be modified during migration.
	if !bytes.Equal(checksum, newClientState.Checksum) {
		return result, errorsmod.Wrapf(types.ErrWasmInvalidContractModification, "expected checksum %s, got %s", hex.EncodeToString(checksum), hex.EncodeToString(newClientState.Checksum))
	}

	return result, nil
}

// WasmInstantiate accepts a message to instantiate a wasm contract, JSON encodes it and calls instantiateContract.
func WasmInstantiate(ctx sdk.Context, k Keeper, vm ibcwasm.WasmEngine, clientID string, cdc codec.BinaryCodec, clientStore storetypes.KVStore, cs *types.ClientState, payload types.InstantiateMessage) error {
	encodedData, err := json.Marshal(payload)
	if err != nil {
		return errorsmod.Wrap(err, "failed to marshal payload for wasm contract instantiation")
	}

	checksum := cs.Checksum
	res, err := k.instantiateContract(ctx, vm, clientID, clientStore, checksum, encodedData)
	if err != nil {
		return errorsmod.Wrap(types.ErrVMError, err.Error())
	}
	if res.Err != "" {
		return errorsmod.Wrap(types.ErrWasmContractCallFailed, res.Err)
	}

	if err = types.CheckResponse(res.Ok); err != nil {
		return errorsmod.Wrapf(err, "checksum (%s)", hex.EncodeToString(cs.Checksum))
	}

	newClientState, err := types.ValidatePostExecutionClientState(clientStore, cdc)
	if err != nil {
		return err
	}

	// Checksum should only be able to be modified during migration.
	if !bytes.Equal(checksum, newClientState.Checksum) {
		return errorsmod.Wrapf(types.ErrWasmInvalidContractModification, "expected checksum %s, got %s", hex.EncodeToString(checksum), hex.EncodeToString(newClientState.Checksum))
	}

	return nil
}

// queryContract calls vm.Query.
func (k Keeper) queryContract(ctx sdk.Context, vm ibcwasm.WasmEngine, clientID string, clientStore storetypes.KVStore, checksum types.Checksum, msg []byte) (*wasmvmtypes.QueryResult, error) {
	sdkGasMeter := ctx.GasMeter()
	multipliedGasMeter := types.NewMultipliedGasMeter(sdkGasMeter, VMGasRegister)
	gasLimit := VMGasRegister.RuntimeGasForContract(ctx)

	env := getEnv(ctx, clientID)

	ctx.GasMeter().ConsumeGas(VMGasRegister.SetupContractCost(true, len(msg)), "Loading CosmWasm module: query")
	resp, gasUsed, err := vm.Query(checksum, env, msg, types.NewStoreAdapter(clientStore), wasmvmAPI, types.NewQueryHandler(ctx, clientID), multipliedGasMeter, gasLimit, types.CostJSONDeserialization)
	VMGasRegister.ConsumeRuntimeGas(ctx, gasUsed)
	return resp, err
}

// callContract calls vm.Sudo with internally constructed gas meter and environment.
func (k Keeper) callContract(ctx sdk.Context, vm ibcwasm.WasmEngine, clientID string, clientStore storetypes.KVStore, checksum types.Checksum, msg []byte) (*wasmvmtypes.ContractResult, error) {
	sdkGasMeter := ctx.GasMeter()
	multipliedGasMeter := types.NewMultipliedGasMeter(sdkGasMeter, VMGasRegister)
	gasLimit := VMGasRegister.RuntimeGasForContract(ctx)

	env := getEnv(ctx, clientID)

	ctx.GasMeter().ConsumeGas(VMGasRegister.SetupContractCost(true, len(msg)), "Loading CosmWasm module: sudo")
	resp, gasUsed, err := vm.Sudo(checksum, env, msg, types.NewStoreAdapter(clientStore), wasmvmAPI, types.NewQueryHandler(ctx, clientID), multipliedGasMeter, gasLimit, types.CostJSONDeserialization)
	VMGasRegister.ConsumeRuntimeGas(ctx, gasUsed)
	return resp, err
}

// instantiateContract calls vm.Instantiate with appropriate arguments.
func (Keeper) instantiateContract(ctx sdk.Context, vm ibcwasm.WasmEngine, clientID string, clientStore storetypes.KVStore, checksum types.Checksum, msg []byte) (*wasmvmtypes.ContractResult, error) {
	sdkGasMeter := ctx.GasMeter()
	multipliedGasMeter := types.NewMultipliedGasMeter(sdkGasMeter, types.VMGasRegister)
	gasLimit := VMGasRegister.RuntimeGasForContract(ctx)

	env := getEnv(ctx, clientID)

	msgInfo := wasmvmtypes.MessageInfo{
		Sender: "",
		Funds:  nil,
	}

	ctx.GasMeter().ConsumeGas(types.VMGasRegister.SetupContractCost(true, len(msg)), "Loading CosmWasm module: instantiate")
	resp, gasUsed, err := vm.Instantiate(checksum, env, msgInfo, msg, types.NewStoreAdapter(clientStore), wasmvmAPI, types.NewQueryHandler(ctx, clientID), multipliedGasMeter, gasLimit, types.CostJSONDeserialization)
	types.VMGasRegister.ConsumeRuntimeGas(ctx, gasUsed)
	return resp, err
}

// getEnv returns the state of the blockchain environment the contract is running on
func getEnv(ctx sdk.Context, contractAddr string) wasmvmtypes.Env {
	chainID := ctx.BlockHeader().ChainID
	height := ctx.BlockHeader().Height

	// safety checks before casting below
	if height < 0 {
		panic(errors.New("block height must never be negative"))
	}
	nsec := ctx.BlockTime().UnixNano()
	if nsec < 0 {
		panic(errors.New("block (unix) time must never be negative "))
	}

	env := wasmvmtypes.Env{
		Block: wasmvmtypes.BlockInfo{
			Height:  uint64(height),
			Time:    wasmvmtypes.Uint64(nsec),
			ChainID: chainID,
		},
		Contract: wasmvmtypes.ContractInfo{
			Address: contractAddr,
		},
	}

	return env
}

func humanizeAddress(canon []byte) (string, uint64, error) {
	return "", 0, errors.New("humanizeAddress not implemented")
}

func canonicalizeAddress(human string) ([]byte, uint64, error) {
	return nil, 0, errors.New("canonicalizeAddress not implemented")
}

func validateAddress(human string) (uint64, error) {
	return 0, errors.New("validateAddress not implemented")
}
