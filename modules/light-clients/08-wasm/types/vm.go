package types

import (
	cosmwasm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var (
	WasmVM *cosmwasm.VM
	// Store key for 08-wasm module, required as a global so that the KV store can be retrieved
	// in the ClientState Initialize function which doesn't have access to the keeper. 
	// The storeKey is used to check the code hash of the contract and determine if the light client
	// is allowed to be instantiated.
	WasmStoreKey  storetypes.StoreKey
	VMGasRegister = NewDefaultWasmGasRegister()
)

type ContractResult interface {
	Validate() bool
	Error() string
}

type contractResult struct {
	IsValid  bool   `json:"is_valid,omitempty"`
	ErrorMsg string `json:"error_msg,omitempty"`
	Data     []byte `json:"data,omitempty"`
}

func (r contractResult) Validate() bool {
	return r.IsValid
}

func (r contractResult) Error() string {
	return r.ErrorMsg
}

type statusQueryResponse struct {
	contractResult
	Status exported.Status `json:"status"`
}

type metadataQueryResponse struct {
	contractResult
	GenesisMetadata []clienttypes.GenesisMetadata `json:"genesis_metadata"`
}

type timestampAtHeightQueryResponse struct {
	contractResult
	Timestamp uint64 `json:"timestamp"`
}

type checkForMisbehaviourExecuteResult struct {
	contractResult
	FoundMisbehaviour bool `json:"found_misbehaviour"`
}

// initContract calls vm.Init with appropriate arguments.
func initContract(ctx sdk.Context, clientStore sdk.KVStore, codeHash []byte) (*wasmvmtypes.Response, error) {
	sdkGasMeter := ctx.GasMeter()
	multipliedGasMeter := NewMultipliedGasMeter(sdkGasMeter, VMGasRegister)
	gasLimit := VMGasRegister.runtimeGasForContract(ctx)

	env := getEnv(ctx)

	msgInfo := wasmvmtypes.MessageInfo{
		Sender: "",
		Funds:  nil,
	}

	initMsg := []byte("{}")
	ctx.GasMeter().ConsumeGas(VMGasRegister.NewContractInstanceCosts(len(initMsg)), "Loading CosmWasm module: instantiate")
	response, gasUsed, err := WasmVM.Instantiate(codeHash, env, msgInfo, initMsg, newStoreAdapter(clientStore), cosmwasm.GoAPI{}, nil, multipliedGasMeter, gasLimit, costJSONDeserialization)
	VMGasRegister.consumeRuntimeGas(ctx, gasUsed)
	return response, err
}

// callContract calls vm.Sudo with internally constructed gas meter and environment.
func callContract(ctx sdk.Context, clientStore sdk.KVStore, codeHash []byte, msg []byte) (*wasmvmtypes.Response, error) {
	sdkGasMeter := ctx.GasMeter()
	multipliedGasMeter := NewMultipliedGasMeter(sdkGasMeter, VMGasRegister)
	gasLimit := VMGasRegister.runtimeGasForContract(ctx)
	env := getEnv(ctx)

	ctx.GasMeter().ConsumeGas(VMGasRegister.InstantiateContractCosts(len(msg)), "Loading CosmWasm module: sudo")
	resp, gasUsed, err := WasmVM.Sudo(codeHash, env, msg, newStoreAdapter(clientStore), cosmwasm.GoAPI{}, nil, multipliedGasMeter, gasLimit, costJSONDeserialization)
	VMGasRegister.consumeRuntimeGas(ctx, gasUsed)
	return resp, err
}

// queryContract calls vm.Query.
func queryContract(ctx sdk.Context, clientStore sdk.KVStore, codeHash []byte, msg []byte) ([]byte, error) {
	sdkGasMeter := ctx.GasMeter()
	multipliedGasMeter := NewMultipliedGasMeter(sdkGasMeter, VMGasRegister)
	gasLimit := VMGasRegister.runtimeGasForContract(ctx)

	env := getEnv(ctx)

	ctx.GasMeter().ConsumeGas(VMGasRegister.InstantiateContractCosts(len(msg)), "Loading CosmWasm module: query")
	resp, gasUsed, err := WasmVM.Query(codeHash, env, msg, newStoreAdapter(clientStore), cosmwasm.GoAPI{}, nil, multipliedGasMeter, gasLimit, costJSONDeserialization)
	VMGasRegister.consumeRuntimeGas(ctx, gasUsed)
	return resp, err
}

// getEnv returns the state of the blockchain environment the contract is running on
func getEnv(ctx sdk.Context) wasmvmtypes.Env {
	chainID := ctx.BlockHeader().ChainID
	height := ctx.BlockHeader().Height

	// safety checks before casting below
	if height < 0 {
		panic("Block height must never be negative")
	}
	nsec := ctx.BlockTime().UnixNano()
	if nsec < 0 {
		panic("Block (unix) time must never be negative ")
	}

	env := wasmvmtypes.Env{
		Block: wasmvmtypes.BlockInfo{
			Height:  uint64(height),
			Time:    uint64(nsec),
			ChainID: chainID,
		},
		Contract: wasmvmtypes.ContractInfo{
			Address: "",
		},
	}

	return env
}
