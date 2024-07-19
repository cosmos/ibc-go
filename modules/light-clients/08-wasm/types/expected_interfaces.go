package types

import (
	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
)

type WasmEngine interface {
	// StoreCode will compile the Wasm code, and store the resulting compiled module
	// as well as the original code. Both can be referenced later via Checksum.
	// This must be done one time for given code, after which it can be
	// instantiated many times, and each instance called many times.
	//
	// Returns both the checksum, as well as the gas cost of compilation (in CosmWasm Gas) or an error.
	StoreCode(code wasmvm.WasmCode, gasLimit uint64) (wasmvmtypes.Checksum, uint64, error)

	// StoreCodeUnchecked will compile the wasm code, and store the resulting pre-compile
	// as well as the original code. Both can be referenced later via checksum
	// This must be done one time for given code, after which it can be
	// instantiated many times, and each instance called many times.
	// It does the same as StoreCode but without the static checks.
	// This allows restoring previous contract code in genesis and state-sync that may have been initially stored under different configuration constraints.
	StoreCodeUnchecked(code wasmvm.WasmCode) (wasmvm.Checksum, error)

	// Instantiate will create a new contract based on the given checksum.
	// We can set the initMsg (contract "genesis") here, and it then receives
	// an account and address and can be invoked (Execute) many times.
	//
	// Storage should be set with a PrefixedKVStore that this code can safely access.
	//
	// Under the hood, we may recompile the wasm, use a cached native compile, or even use a cached instance
	// for performance.
	Instantiate(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		info wasmvmtypes.MessageInfo,
		initMsg []byte,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.ContractResult, uint64, error)

	// Query allows a client to execute a contract-specific query. If the result is not empty, it should be
	// valid json-encoded data to return to the client.
	// The meaning of path and data can be determined by the code. Path is the suffix of the abci.QueryRequest.Path
	Query(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		queryMsg []byte,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.QueryResult, uint64, error)

	// Migrate will migrate an existing contract to a new code binary.
	// This takes storage of the data from the original contract and the checksum of the new contract that should
	// replace it. This allows it to run a migration step if needed, or return an error if unable to migrate
	// the given data.
	//
	// MigrateMsg has some data on how to perform the migration.
	Migrate(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		migrateMsg []byte,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.ContractResult, uint64, error)

	// Sudo allows native Go modules to make privileged (sudo) calls on the contract.
	// The contract can expose entry points that cannot be triggered by any transaction, but only via
	// native Go modules, and delegate the access control to the system.
	//
	// These work much like Migrate (same scenario) but allows custom apps to extend the privileged entry points
	// without forking cosmwasm-vm.
	Sudo(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		sudoMsg []byte,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.ContractResult, uint64, error)

	// GetCode will load the original wasm code for the given checksum.
	// This will only succeed if that checksum was previously returned from
	// a call to Create.
	//
	// This can be used so that the (short) checksum is stored in the iavl tree
	// and the larger binary blobs (wasm and pre-compiles) are all managed by the
	// rust library
	GetCode(checksum wasmvm.Checksum) (wasmvm.WasmCode, error)

	// Pin pins a code to an in-memory cache, such that is
	// always loaded quickly when executed.
	// Pin is idempotent.
	Pin(checksum wasmvm.Checksum) error

	// Unpin removes the guarantee of a contract to be pinned (see Pin).
	// After calling this, the code may or may not remain in memory depending on
	// the implementor's choice.
	// Unpin is idempotent.
	Unpin(checksum wasmvm.Checksum) error
}

type QueryRouter interface {
	// Route returns the GRPCQueryHandler for a given query route path or nil
	// if not found
	Route(path string) baseapp.GRPCQueryHandler
}
