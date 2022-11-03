package types

import (
	"bytes"
	"crypto/sha256"
	"strings"

	cosmwasm "github.com/CosmWasm/wasmvm"
	"github.com/CosmWasm/wasmvm/types"
	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	committypes "github.com/cosmos/ibc-go/v5/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v5/modules/core/exported"
)

// TODO: figure out better handling for the gas settings. ideally these should be in the
// 28-wasm module and handled as params
const GasMultiplier uint64 = 100
const maxGasLimit = uint64(0x7FFFFFFFFFFFFFFF)

var WasmVM *cosmwasm.VM
var WasmVal *WasmValidator

var _ exported.ClientState = (*ClientState)(nil)

// generalize this maybe?
type queryResponse struct {
	ProofSpecs      []*ics23.ProofSpec            `json:"proof_specs,omitempty"`
	Height          clienttypes.Height            `json:"height,omitempty"`
	GenesisMetadata []clienttypes.GenesisMetadata `json:"genesis_metadata,omitempty"`
	Result          contractResult                `json:"result,omitempty"`
	Root            committypes.MerkleRoot        `json:"root,omitempty"`
	Timestamp       uint64                        `json:"timestamp,omitempty"`
	Status          exported.Status               `json:"status,omitempty"`
}

type contractResult struct {
	IsValid  bool   `json:"is_valid,omitempty"`
	ErrorMsg string `json:"err_msg,omitempty"`
}

type VMConfig struct {
	DataDir           string
	SupportedFeatures []string
	MemoryLimitMb     uint32
	PrintDebug        bool
	CacheSizeMb       uint32
}

// TODO: Move this into the 28-wasm keeper
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

func CreateVM(vmConfig *VMConfig, validationConfig *ValidationConfig) {
	supportedFeatures := strings.Join(vmConfig.SupportedFeatures, ",")

	vm, err := cosmwasm.NewVM(vmConfig.DataDir, supportedFeatures, vmConfig.MemoryLimitMb, vmConfig.PrintDebug, vmConfig.CacheSizeMb)
	if err != nil {
		panic(err)
	}

	wasmValidator, err := NewWasmValidator(validationConfig, func() (*cosmwasm.VM, error) {
		return cosmwasm.NewVM(vmConfig.DataDir, supportedFeatures, vmConfig.MemoryLimitMb, vmConfig.PrintDebug, vmConfig.CacheSizeMb)
	})
	if err != nil {
		panic(err)
	}

	WasmVM = vm
	WasmVal = wasmValidator
}

func SaveClientStateIntoWasmStorage(ctx sdk.Context, cdc codec.BinaryCodec, store sdk.KVStore, c *ClientState) (*types.Response, error) {
	msg := clienttypes.MustMarshalClientState(cdc, c)
	return callContract(c.CodeId, ctx, store, msg)
}

func PushNewWasmCode(store sdk.KVStore, c *ClientState, code []byte) error {
	// check to see if the store has a code with the same code id
	codeHash := generateWasmCodeHash(code)
	codeIDKey := CodeID(codeHash)
	if store.Has(codeIDKey) {
		return ErrWasmCodeExists
	}

	// run the code through the wasm light client validation process
	if isValidWasmCode, err := WasmVal.validateWasmCode(code); err != nil {
		return sdkerrors.Wrapf(ErrWasmCodeValidation, "unable to validate wasm code: %s", err)
	} else if !isValidWasmCode {
		return ErrWasmInvalidCode
	}

	// create the code in the vm
	// TODO: do we need to check and make sure there
	// is no code with the same hash?
	codeID, err := WasmVM.Create(code)
	if err != nil {
		return ErrWasmInvalidCode
	}

	// safety check to assert that code id returned by WasmVM equals to code hash
	if !bytes.Equal(codeID, codeHash) {
		return ErrWasmInvalidCodeID
	}

	store.Set(codeIDKey, code)
	return nil
}

// Calls vm.Init with appropriate arguments
// TODO: Move this into a public method on the 28-wasm keeper
func initContract(codeID []byte, ctx sdk.Context, store sdk.KVStore, msg []byte) (*types.Response, error) {
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
	// mockFailureAPI := *api.NewMockFailureAPI()
	// mockQuerier := api.MockQuerier{}

	desercost := types.UFraction{Numerator: 0, Denominator: 1}
	response, _, err := WasmVM.Instantiate(codeID, env, msgInfo, msg, store, cosmwasm.GoAPI{}, nil, gasMeter, gasMeter.Limit(), desercost)
	return response, err
}

// Calls vm.Execute with internally constructed Gas meter and environment
// TODO: Move this into a public method on the 28-wasm keeper
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

	return callContractWithEnvAndMeter(codeID, &ctx, store, env, gasMeter, msg)
}

// Calls vm.Execute with supplied environment and gas meter
// TODO: Move this into a private method on the 28-wasm keeper
func callContractWithEnvAndMeter(codeID cosmwasm.Checksum, ctx *sdk.Context, store cosmwasm.KVStore, env types.Env, gasMeter sdk.GasMeter, msg []byte) (*types.Response, error) {
	msgInfo := types.MessageInfo{
		Sender: "",
		Funds:  nil,
	}
	// TODO: fix this
	// mockFailureAPI := *api.NewMockFailureAPI()
	// mockQuerier := api.MockQuerier{}
	desercost := types.UFraction{Numerator: 1, Denominator: 1}
	resp, gasUsed, err := WasmVM.Execute(codeID, env, msgInfo, msg, store, cosmwasm.GoAPI{}, nil, nil, gasMeter.Limit(), desercost)
	if ctx != nil {
		consumeGas(*ctx, gasUsed)
	}
	return resp, err
}

// TODO: Move this into a public method on the 28-wasm keeper
func queryContractWithStore(codeID cosmwasm.Checksum, store cosmwasm.KVStore, msg []byte) ([]byte, error) {
	// TODO: fix this
	// mockEnv := api.MockEnv()
	// mockGasMeter := api.NewMockGasMeter(1)
	// mockFailureAPI := *api.NewMockFailureAPI()
	// mockQuerier := api.MockQuerier{}
	// TODO: figure out what this is for
	desercost := types.UFraction{Numerator: 1, Denominator: 1}
	resp, _, err := WasmVM.Query(codeID, types.Env{}, msg, store, cosmwasm.GoAPI{}, nil, nil, maxGasLimit, desercost)
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

func generateWasmCodeHash(code []byte) []byte {
	hash := sha256.Sum256(code)
	return hash[:]
}
