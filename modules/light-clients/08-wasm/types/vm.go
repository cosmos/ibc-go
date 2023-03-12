package types

import (
	cosmwasm "github.com/CosmWasm/wasmvm"
	"github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

const GasMultiplier uint64 = 100
const maxGasLimit = uint64(0x7FFFFFFFFFFFFFFF)

var WasmVM *cosmwasm.VM

type queryResponse struct {
	Status          exported.Status               `json:"status,omitempty"`
}

type ClientCreateRequest struct {
	ClientCreateRequest ClientState `json:"client_create_request,omitempty"`
}

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

type clientStateCallResponse struct {
	Me                *ClientState    `json:"me,omitempty"`
	NewConsensusState *ConsensusState `json:"new_consensus_state,omitempty"`
	NewClientState    *ClientState    `json:"new_client_state,omitempty"`
	Result            contractResult  `json:"result,omitempty"`
}

func (r *clientStateCallResponse) resetImmutables(c *ClientState) {
	if r.Me != nil {
		r.Me.CodeId = c.CodeId
	}

	if r.NewConsensusState != nil {
		r.NewConsensusState.CodeId = c.CodeId
	}

	if r.NewClientState != nil {
		r.NewClientState.CodeId = c.CodeId
	}
}

func (r clientStateCallResponse) Validate() bool {
	return r.Result.Validate()
}

func (r clientStateCallResponse) Error() string {
	return r.Result.Error()
}

// Calls vm.Init with appropriate arguments
func initContract(codeID []byte, ctx sdk.Context, store sdk.KVStore) (*types.Response, error) {
	gasMeter := ctx.GasMeter()
	chainID := ctx.BlockHeader().ChainID
	height := ctx.BlockHeader().Height
	// safety checks before casting below
	if height < 0 {
		panic("Block height must never be negative")
	}
	sec := ctx.BlockTime().Unix()
	if sec < 0 {
		panic("Block (unix) time must never be negative ")
	}
	env := types.Env{
		Block: types.BlockInfo{
			Height:  uint64(height),
			Time:    uint64(sec),
			ChainID: chainID,
		},
		Contract: types.ContractInfo{
			Address: "",
		},
	}

	msgInfo := types.MessageInfo{
		Sender: "",
		Funds:  nil,
	}

	desercost := types.UFraction{Numerator: 0, Denominator: 1}
	response, _, err := WasmVM.Instantiate(codeID, env, msgInfo, []byte("{}"), store, cosmwasm.GoAPI{}, nil, gasMeter, gasMeter.Limit(), desercost)
	return response, err
}

// Calls vm.Execute with internally constructed Gas meter and environment
func callContract(codeID []byte, ctx sdk.Context, store sdk.KVStore, msg []byte) (*types.Response, error) {
	gasMeter := ctx.GasMeter()
	chainID := ctx.BlockHeader().ChainID
	height := ctx.BlockHeader().Height
	// safety checks before casting below
	if height < 0 {
		panic("Block height must never be negative")
	}
	sec := ctx.BlockTime().Unix()
	if sec < 0 {
		panic("Block (unix) time must never be negative ")
	}
	env := types.Env{
		Block: types.BlockInfo{
			Height:  uint64(height),
			Time:    uint64(sec),
			ChainID: chainID,
		},
		Contract: types.ContractInfo{
			Address: "",
		},
	}

	return callContractWithEnvAndMeter(codeID, ctx, store, env, gasMeter, msg)
}

// Calls vm.Execute with supplied environment and gas meter
func callContractWithEnvAndMeter(codeID cosmwasm.Checksum, ctx sdk.Context, store sdk.KVStore, env types.Env, gasMeter sdk.GasMeter, msg []byte) (*types.Response, error) {
	msgInfo := types.MessageInfo{
		Sender: "",
		Funds:  nil,
	}
	desercost := types.UFraction{Numerator: 1, Denominator: 1}
	resp, gasUsed, err := WasmVM.Execute(codeID, env, msgInfo, msg, store, cosmwasm.GoAPI{}, nil, gasMeter, gasMeter.Limit(), desercost)
	if &ctx != nil {
		consumeGas(ctx, gasUsed)
	}
	return resp, err
}

// Call vm.Query
func queryContractWithStore(codeID cosmwasm.Checksum, ctx sdk.Context, store sdk.KVStore, msg []byte) ([]byte, error) {
	gasMeter := ctx.GasMeter()
	chainID := ctx.BlockHeader().ChainID
	height := ctx.BlockHeader().Height
	// safety checks before casting below
	if height < 0 {
		panic("Block height must never be negative")
	}
	sec := ctx.BlockTime().Unix()
	if sec < 0 {
		panic("Block (unix) time must never be negative ")
	}
	env := types.Env{
		Block: types.BlockInfo{
			Height:  uint64(height),
			Time:    uint64(sec),
			ChainID: chainID,
		},
		Contract: types.ContractInfo{
			Address: "",
		},
	}
	desercost := types.UFraction{Numerator: 1, Denominator: 1}
	resp, _, err := WasmVM.Query(codeID, env, msg, store, cosmwasm.GoAPI{}, nil, gasMeter, gasMeter.Limit(), desercost)
	return resp, err
}

func consumeGas(ctx sdk.Context, gas uint64) {
	consumed := gas / GasMultiplier
	ctx.GasMeter().ConsumeGas(consumed, "wasm contract")
	// throw OutOfGas error if we ran out (got exactly to zero due to better limit enforcing)
	if ctx.GasMeter().IsOutOfGas() {
		panic(sdk.ErrorOutOfGas{Descriptor: "Wasmer function execution"})
	}
}
