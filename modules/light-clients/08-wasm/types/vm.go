package types

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

var VMGasRegister = NewDefaultWasmGasRegister()

// initContract calls vm.Init with appropriate arguments.
func initContract(ctx sdk.Context, clientStore storetypes.KVStore, codeHash []byte, msg []byte) (*wasmvmtypes.Response, error) {
	sdkGasMeter := ctx.GasMeter()
	multipliedGasMeter := NewMultipliedGasMeter(sdkGasMeter, VMGasRegister)
	gasLimit := VMGasRegister.runtimeGasForContract(ctx)

	clientID, err := getClientID(clientStore)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to retrieve clientID for wasm contract instantiation")
	}
	env := getEnv(ctx, clientID)

	msgInfo := wasmvmtypes.MessageInfo{
		Sender: "",
		Funds:  nil,
	}

	ctx.GasMeter().ConsumeGas(VMGasRegister.NewContractInstanceCosts(len(msg)), "Loading CosmWasm module: instantiate")
	response, gasUsed, err := ibcwasm.GetVM().Instantiate(codeHash, env, msgInfo, msg, newStoreAdapter(clientStore), wasmvm.GoAPI{}, nil, multipliedGasMeter, gasLimit, costJSONDeserialization)
	VMGasRegister.consumeRuntimeGas(ctx, gasUsed)
	return response, err
}

// callContract calls vm.Sudo with internally constructed gas meter and environment.
func callContract(ctx sdk.Context, clientStore storetypes.KVStore, codeHash []byte, msg []byte) (*wasmvmtypes.Response, error) {
	sdkGasMeter := ctx.GasMeter()
	multipliedGasMeter := NewMultipliedGasMeter(sdkGasMeter, VMGasRegister)
	gasLimit := VMGasRegister.runtimeGasForContract(ctx)

	clientID, err := getClientID(clientStore)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to retrieve clientID for wasm contract call")
	}
	env := getEnv(ctx, clientID)

	ctx.GasMeter().ConsumeGas(VMGasRegister.InstantiateContractCosts(len(msg)), "Loading CosmWasm module: sudo")
	resp, gasUsed, err := ibcwasm.GetVM().Sudo(codeHash, env, msg, newStoreAdapter(clientStore), wasmvm.GoAPI{}, nil, multipliedGasMeter, gasLimit, costJSONDeserialization)
	VMGasRegister.consumeRuntimeGas(ctx, gasUsed)
	return resp, err
}

// queryContract calls vm.Query.
func queryContract(ctx sdk.Context, clientStore storetypes.KVStore, codeHash []byte, msg []byte) ([]byte, error) {
	sdkGasMeter := ctx.GasMeter()
	multipliedGasMeter := NewMultipliedGasMeter(sdkGasMeter, VMGasRegister)
	gasLimit := VMGasRegister.runtimeGasForContract(ctx)

	clientID, err := getClientID(clientStore)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to retrieve clientID for wasm contract query")
	}
	env := getEnv(ctx, clientID)

	ctx.GasMeter().ConsumeGas(VMGasRegister.InstantiateContractCosts(len(msg)), "Loading CosmWasm module: query")
	resp, gasUsed, err := ibcwasm.GetVM().Query(codeHash, env, msg, newStoreAdapter(clientStore), wasmvm.GoAPI{}, nil, multipliedGasMeter, gasLimit, costJSONDeserialization)
	VMGasRegister.consumeRuntimeGas(ctx, gasUsed)
	return resp, err
}

// wasmInstantiate accepts a message to instantiate a wasm contract, JSON encodes it and calls initContract.
func wasmInstantiate(ctx sdk.Context, clientStore storetypes.KVStore, cs *ClientState, payload InstantiateMessage) error {
	encodedData, err := json.Marshal(payload)
	if err != nil {
		return errorsmod.Wrap(err, "failed to marshal payload for wasm contract instantiation")
	}
	_, err = initContract(ctx, clientStore, cs.CodeHash, encodedData)
	if err != nil {
		return errorsmod.Wrap(ErrWasmContractCallFailed, err.Error())
	}
	return nil
}

// wasmSudo calls the contract with the given payload and returns the result.
// wasmSudo returns an error if:
// - the payload cannot be marshaled to JSON
// - the contract call returns an error
// - the response of the contract call contains non-empty messages
// - the response of the contract call contains non-empty events
// - the response of the contract call contains non-empty attributes
// - the data bytes of the response cannot be unmarshaled into the result type
func wasmSudo[T ContractResult](ctx sdk.Context, clientStore storetypes.KVStore, cs *ClientState, payload SudoMsg) (T, error) {
	var result T

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return result, errorsmod.Wrap(err, "failed to marshal payload for wasm execution")
	}

	resp, err := callContract(ctx, clientStore, cs.CodeHash, encodedData)
	if err != nil {
		return result, errorsmod.Wrap(ErrWasmContractCallFailed, err.Error())
	}

	// Only allow Data to flow back to us. SubMessages, Events and Attributes are not allowed.
	if len(resp.Messages) > 0 {
		return result, errorsmod.Wrapf(ErrWasmSubMessagesNotAllowed, "code hash (%s)", hex.EncodeToString(cs.CodeHash))
	}
	if len(resp.Events) > 0 {
		return result, errorsmod.Wrapf(ErrWasmEventsNotAllowed, "code hash (%s)", hex.EncodeToString(cs.CodeHash))
	}
	if len(resp.Attributes) > 0 {
		return result, errorsmod.Wrapf(ErrWasmAttributesNotAllowed, "code hash (%s)", hex.EncodeToString(cs.CodeHash))
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return result, errorsmod.Wrap(ErrWasmInvalidResponseData, err.Error())
	}
	return result, nil
}

// wasmQuery queries the contract with the given payload and returns the result.
// wasmQuery returns an error if:
// - the payload cannot be marshaled to JSON
// - the contract query returns an error
// - the data bytes of the response cannot be unmarshal into the result type
func wasmQuery[T ContractResult](ctx sdk.Context, clientStore storetypes.KVStore, cs *ClientState, payload QueryMsg) (T, error) {
	var result T

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return result, errorsmod.Wrap(err, "failed to marshal payload for wasm query")
	}

	resp, err := queryContract(ctx, clientStore, cs.CodeHash, encodedData)
	if err != nil {
		return result, errorsmod.Wrap(ErrWasmContractCallFailed, err.Error())
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return result, errorsmod.Wrapf(ErrWasmInvalidResponseData, "failed to unmarshal result of wasm query: %v", err)
	}

	return result, nil
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
			Time:    uint64(nsec),
			ChainID: chainID,
		},
		Contract: wasmvmtypes.ContractInfo{
			Address: contractAddr,
		},
	}

	return env
}
