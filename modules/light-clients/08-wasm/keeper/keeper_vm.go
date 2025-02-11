//go:build cgo && !nolink_libwasmvm

package keeper

import (
	"errors"
	"fmt"
	"strings"

	wasmvm "github.com/CosmWasm/wasmvm/v2"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

// NewKeeperWithVM creates a new Keeper instance with the provided Wasm VM.
// This constructor function is meant to be used when the chain uses x/wasm
// and the same Wasm VM instance should be shared with it.
func NewKeeperWithVM(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	clientKeeper types.ClientKeeper,
	authority string,
	vm types.WasmEngine,
	queryRouter types.QueryRouter,
	opts ...Option,
) Keeper {
	if clientKeeper == nil {
		panic(errors.New("client keeper must not be nil"))
	}

	if queryRouter == nil {
		panic(errors.New("query router must not be nil"))
	}

	if vm == nil {
		panic(errors.New("wasm VM must not be nil"))
	}

	if storeService == nil {
		panic(errors.New("store service must not be nil"))
	}

	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	sb := collections.NewSchemaBuilder(storeService)

	keeper := &Keeper{
		cdc:          cdc,
		vm:           vm,
		checksums:    collections.NewKeySet(sb, types.ChecksumsKey, "checksums", collections.BytesKey),
		storeService: storeService,
		clientKeeper: clientKeeper,
		authority:    authority,
	}

	_, err := sb.Build()
	if err != nil {
		panic(err)
	}

	// set query plugins to ensure there is a non-nil query plugin
	// regardless of what options the user provides
	keeper.setQueryPlugins(NewDefaultQueryPlugins(queryRouter))

	for _, opt := range opts {
		opt.apply(keeper)
	}

	return *keeper
}

// NewKeeperWithConfig creates a new Keeper instance with the provided Wasm configuration.
// This constructor function is meant to be used when the chain does not use x/wasm
// and a Wasm VM needs to be instantiated using the provided parameters.
func NewKeeperWithConfig(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	clientKeeper types.ClientKeeper,
	authority string,
	wasmConfig types.WasmConfig,
	queryRouter types.QueryRouter,
	opts ...Option,
) Keeper {
	vm, err := wasmvm.NewVM(wasmConfig.DataDir, wasmConfig.SupportedCapabilities, types.ContractMemoryLimit, wasmConfig.ContractDebugMode, types.MemoryCacheSize)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate new Wasm VM instance: %v", err))
	}

	return NewKeeperWithVM(cdc, storeService, clientKeeper, authority, vm, queryRouter, opts...)
}
