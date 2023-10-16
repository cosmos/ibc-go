package types

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
)

const DefaultGasUsed = uint64(1)

var _ WasmEngine = (*MockWasmEngine)(nil)

type QueryFn func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error)

func NewMockWasmEngine() *MockWasmEngine {
	m := &MockWasmEngine{
		queryCallbacks: map[string]QueryFn{},
	}

	// all possible fields that can be passed to the query fn should be specified here.
	allQueryTypes := []any{
		StatusMsg{},
		ExportMetadataMsg{},
		TimestampAtHeightMsg{},
		VerifyClientMessageMsg{},
		CheckForMisbehaviourMsg{},
	}

	for _, msgType := range allQueryTypes {
		typeName := reflect.TypeOf(msgType).Name()
		m.queryCallbacks[typeName] = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
			panic(fmt.Errorf("no callback specified for type %s", typeName))
		}
	}

	return m
}

// RegisterQueryCallback registers a callback for a specific message type.
func (m *MockWasmEngine) RegisterQueryCallback(queryMessage any, fn QueryFn) {
	typeName := reflect.TypeOf(queryMessage).Name()
	if _, found := m.queryCallbacks[typeName]; !found {
		panic(fmt.Errorf("unexpected argument of type %s passed", typeName))
	}
	m.queryCallbacks[typeName] = fn
}

// MockWasmEngine implements types.WasmEngine for testing purpose. One or multiple messages can be stubbed.
// Without a stub function a panic is thrown.
// ref: https://github.com/CosmWasm/wasmd/blob/v0.42.0/x/wasm/keeper/wasmtesting/mock_engine.go#L19
type MockWasmEngine struct {
	StoreCodeFn   func(codeID wasmvm.WasmCode) (wasmvm.Checksum, error)
	InstantiateFn func(codeID wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error)
	SudoFn        func(codeID wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error)
	GetCodeFn     func(codeID wasmvm.Checksum) (wasmvm.WasmCode, error)
	PinFn         func(checksum wasmvm.Checksum) error

	// queryCallbacks contains a mapping of queryMsg field type name to callback function.
	queryCallbacks map[string]QueryFn
}

// StoreCode implements the WasmEngine interface.
func (m *MockWasmEngine) StoreCode(codeID wasmvm.WasmCode) (wasmvm.Checksum, error) {
	if m.StoreCodeFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.StoreCodeFn(codeID)
}

// Instantiate implements the WasmEngine interface.
func (m *MockWasmEngine) Instantiate(codeID wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	if m.InstantiateFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.InstantiateFn(codeID, env, info, initMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

// Query implements the WasmEngine interface.
func (m *MockWasmEngine) Query(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
	msgTypeName := getQueryMsgPayloadTypeName(queryMsg)

	callbackFn, ok := m.queryCallbacks[msgTypeName]
	if !ok {
		panic(fmt.Errorf("no callback specified for %s", msgTypeName))
	}

	return callbackFn(codeID, env, queryMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

// Sudo implements the WasmEngine interface.
func (m *MockWasmEngine) Sudo(codeID wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	if m.SudoFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.SudoFn(codeID, env, sudoMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

// GetCode implements the WasmEngine interface.
func (m *MockWasmEngine) GetCode(codeID wasmvm.Checksum) (wasmvm.WasmCode, error) {
	if m.GetCodeFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.GetCodeFn(codeID)
}

// Pin implements the WasmEngine interface.
func (m *MockWasmEngine) Pin(checksum wasmvm.Checksum) error {
	if m.PinFn == nil {
		panic("mock engine is not properly initialized")
	}
	return m.PinFn(checksum)
}

func upperCaseFirstChar(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

// getQueryMsgPayloadTypeName extracts the name of the struct that is populated.
// this value is used as a key to map to a callback function to handle that message type.
func getQueryMsgPayloadTypeName(queryMsgBz []byte) string {
	payload := queryMsg{}
	if err := json.Unmarshal(queryMsgBz, &payload); err != nil {
		panic(err)
	}

	var payloadField any
	if payload.Status != nil {
		payloadField = *payload.Status
	}

	if payload.CheckForMisbehaviour != nil {
		payloadField = *payload.CheckForMisbehaviour
	}

	if payload.ExportMetadata != nil {
		payloadField = *payload.ExportMetadata
	}

	if payload.TimestampAtHeight != nil {
		payloadField = *payload.TimestampAtHeight
	}

	if payload.VerifyClientMessage != nil {
		payloadField = *payload.VerifyClientMessage
	}

	if payloadField == nil {
		panic(fmt.Errorf("failed to extract valid query message from bytes: %s", string(queryMsgBz)))
	}

	return upperCaseFirstChar(reflect.TypeOf(payloadField).Name())
}
