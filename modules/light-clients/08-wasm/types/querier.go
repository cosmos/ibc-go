package types

import (
	"encoding/json"
	"fmt"
	"slices"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

var (
	_ wasmvmtypes.Querier = (*QueryHandler)(nil)

	queryPlugins = NewDefaultQueryPlugins()
)

// QueryHandler is a wrapper around the sdk.Context and the CallerID that calls
// into the query plugins.
type QueryHandler struct {
	Ctx      sdk.Context
	CallerID string
}

// NewQueryHandler returns a default querier that can be used in the contract.
func NewQueryHandler(ctx sdk.Context, callerID string) *QueryHandler {
	return &QueryHandler{
		Ctx:      ctx,
		CallerID: callerID,
	}
}

// GasConsumed implements the wasmvmtypes.Querier interface.
func (q *QueryHandler) GasConsumed() uint64 {
	return VMGasRegister.ToWasmVMGas(q.Ctx.GasMeter().GasConsumed())
}

// Query implements the wasmvmtypes.Querier interface.
func (q *QueryHandler) Query(request wasmvmtypes.QueryRequest, gasLimit uint64) ([]byte, error) {
	sdkGas := VMGasRegister.FromWasmVMGas(gasLimit)

	subCtx, _ := q.Ctx.WithGasMeter(storetypes.NewGasMeter(sdkGas)).CacheContext()

	// make sure we charge the higher level context even on panic
	defer func() {
		q.Ctx.GasMeter().ConsumeGas(subCtx.GasMeter().GasConsumed(), "contract sub-query")
	}()

	res, err := GetQueryPlugins().HandleQuery(subCtx, q.CallerID, request)
	if err == nil {
		return res, nil
	}

	Logger(q.Ctx).Debug("Redacting query error", "cause", err)
	return nil, redactError(err)
}

type (
	CustomQuerier   func(ctx sdk.Context, request json.RawMessage) ([]byte, error)
	StargateQuerier func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error)
)

// QueryPlugins is a list of query handlers that can be used to extend the default querier.
type QueryPlugins struct {
	Custom   CustomQuerier
	Stargate StargateQuerier
}

// Merge merges the query plugin with a provided one.
func (e QueryPlugins) Merge(x *QueryPlugins) QueryPlugins {
	// only update if this is non-nil and then only set values
	if x == nil {
		return e
	}

	if x.Custom != nil {
		e.Custom = x.Custom
	}

	if x.Stargate != nil {
		e.Stargate = x.Stargate
	}

	return e
}

func (e QueryPlugins) HandleQuery(ctx sdk.Context, caller string, request wasmvmtypes.QueryRequest) ([]byte, error) {
	if request.Stargate != nil {
		return e.Stargate(ctx, request.Stargate)
	}

	if request.Custom != nil {
		return e.Custom(ctx, request.Custom)
	}

	return nil, wasmvmtypes.UnsupportedRequest{Kind: "Unsupported query request"}
}

// SetQueryPlugins sets the current query plugins
func SetQueryPlugins(plugins *QueryPlugins) {
	queryPlugins = plugins
}

// GetQueryPlugins returns the current query plugins
func GetQueryPlugins() *QueryPlugins {
	return queryPlugins
}

// NewDefaultQueryPlugins returns the default set of query plugins
func NewDefaultQueryPlugins() *QueryPlugins {
	return &QueryPlugins{
		Custom:   RejectCustomQuerier(),
		Stargate: AcceptListStargateQuerier([]string{}),
	}
}

// AcceptListStargateQuerier allows all queries that are in the accept list provided and in the default accept list.
// This function returns protobuf encoded responses in bytes.
func AcceptListStargateQuerier(accepted []string) func(sdk.Context, *wasmvmtypes.StargateQuery) ([]byte, error) {
	return func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
		// A default list of accepted queries can be added here.
		// accepted = append(defaultAcceptList, accepted...)

		isAccepted := slices.Contains(accepted, request.Path)
		if !isAccepted {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", request.Path)}
		}

		route := ibcwasm.GetQueryRouter().Route(request.Path)
		if route == nil {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("No route to query '%s'", request.Path)}
		}

		res, err := route(ctx, &abci.RequestQuery{
			Data: request.Data,
			Path: request.Path,
		})
		if err != nil {
			return nil, err
		}
		if res == nil || res.Value == nil {
			return nil, errorsmod.Wrap(ErrInvalid, "Query response is empty")
		}

		return res.Value, nil
	}
}

// RejectCustomQuerier rejects all custom queries
func RejectCustomQuerier() func(sdk.Context, json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "Custom queries are not allowed"}
	}
}

// Wasmd Issue [#759](https://github.com/CosmWasm/wasmd/issues/759)
// Don't return error string for worries of non-determinism
func redactError(err error) error {
	// Do not redact system errors
	// SystemErrors must be created in 08-wasm and we can ensure determinism
	if wasmvmtypes.ToSystemError(err) != nil {
		return err
	}

	codespace, code, _ := errorsmod.ABCIInfo(err, false)
	return fmt.Errorf("codespace: %s, code: %d", codespace, code)
}
