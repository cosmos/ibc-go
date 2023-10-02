package mock

import (
	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

var _ types.WasmEngine = &MockWasmEngine{}

// MockWasmEngine implements types.WasmEngine for testing purpose. One or multiple messages can be stubbed.
// Without a stub function a panic is thrown.
type MockWasmEngine struct {
	StoreCodeFn   func(codeID wasmvm.WasmCode) (wasmvm.Checksum, error)
	InstantiateFn func(codeID wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error)
	QueryFn       func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error)
	SudoFn        func(codeID wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error)
	GetCodeFn     func(codeID wasmvm.Checksum) (wasmvm.WasmCode, error)
	PinFn         func(checksum wasmvm.Checksum) error
}

func (m *MockWasmEngine) StoreCode(codeID wasmvm.WasmCode) (wasmvm.Checksum, error) {
	if m.StoreCodeFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.StoreCodeFn(codeID)
}

func (m *MockWasmEngine) Instantiate(codeID wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	if m.InstantiateFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.InstantiateFn(codeID, env, info, initMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (m *MockWasmEngine) Query(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
	if m.QueryFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.QueryFn(codeID, env, queryMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (m *MockWasmEngine) Sudo(codeID wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	if m.SudoFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.SudoFn(codeID, env, sudoMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (m *MockWasmEngine) GetCode(codeID wasmvm.Checksum) (wasmvm.WasmCode, error) {
	if m.GetCodeFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.GetCodeFn(codeID)
}

func (m *MockWasmEngine) Pin(checksum wasmvm.Checksum) error {
	if m.PinFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.PinFn(checksum)
}
