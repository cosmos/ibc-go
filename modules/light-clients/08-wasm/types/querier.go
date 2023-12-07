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
	_ wasmvmtypes.Querier = (*DefaultQuerier)(nil)

	defaultAcceptList = []string{}
	QuerierPlugins    = NewDefaultQueryPlugins()
)

type DefaultQuerier struct {
	Ctx      sdk.Context
	CallerID string
}

// NewDefaultQuerier returns a default querier that can be used in the contract.
func NewQueryHandler(ctx sdk.Context, callerID string) *DefaultQuerier {
	return &DefaultQuerier{
		Ctx:      ctx,
		CallerID: callerID,
	}
}

// GasConsumed implements the wasmvmtypes.Querier interface.
func (q *DefaultQuerier) GasConsumed() uint64 {
	return VMGasRegister.ToWasmVMGas(q.Ctx.GasMeter().GasConsumed())
}

// Query implements the wasmvmtypes.Querier interface.
func (q *DefaultQuerier) Query(request wasmvmtypes.QueryRequest, gasLimit uint64) ([]byte, error) {
	sdkGas := VMGasRegister.FromWasmVMGas(gasLimit)

	subCtx, _ := q.Ctx.WithGasMeter(storetypes.NewGasMeter(sdkGas)).CacheContext()

	// make sure we charge the higher level context even on panic
	defer func() {
		q.Ctx.GasMeter().ConsumeGas(subCtx.GasMeter().GasConsumed(), "contract sub-query")
	}()

	if request.Stargate != nil {
		return GetQueryPlugins().Stargate(subCtx, request.Stargate)
	}

	if request.Custom != nil {
		return GetQueryPlugins().Custom(subCtx, request.Custom)
	}

	return nil, wasmvmtypes.UnsupportedRequest{Kind: "Unsupported query request"}
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
func (e QueryPlugins) Merge(o *QueryPlugins) QueryPlugins {
	// only update if this is non-nil and then only set values
	if o == nil {
		return e
	}

	if o.Custom != nil {
		e.Custom = o.Custom
	}

	if o.Stargate != nil {
		e.Stargate = o.Stargate
	}

	return e
}

// SetQueryPlugins sets the current query plugins
func SetQueryPlugins(plugins *QueryPlugins) {
	QuerierPlugins = plugins
}

// GetQueryPlugins returns the current query plugins
func GetQueryPlugins() *QueryPlugins {
	return QuerierPlugins
}

// NewDefaultQueryPlugins returns the default set of query plugins
func NewDefaultQueryPlugins() *QueryPlugins {
	return &QueryPlugins{
		Custom:   RejectCustomQuerier(),
		Stargate: AcceptListStargateQuerier([]string{}),
	}
}

// AcceptListStargateQuerier allows all stargate queries in the GRPCQueryAllowList
func AcceptListStargateQuerier(accepted []string) func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
	return func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
		accepted = append(defaultAcceptList, accepted...)

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
			return nil, errorsmod.Wrap(ErrInvalid, "query response is empty")
		}

		return res.Value, nil
	}
}

// RejectCustomQuerier rejects all custom queries
func RejectCustomQuerier() func(sdk.Context, json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "Custom queries are disabled"}
	}
}
