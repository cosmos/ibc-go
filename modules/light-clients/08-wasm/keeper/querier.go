package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	"cosmossdk.io/core/appmodule"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/api/cometbft/abci/v1"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

/*
`queryHandler` is a contextual querier which references the global `ibcwasm.QueryPluginsI`
to handle queries. The global `ibcwasm.QueryPluginsI` points to a `types.QueryPlugins` which
contains two sub-queriers: `types.CustomQuerier` and `types.StargateQuerier`. These sub-queriers
can be replaced by the user through the options api in the keeper.

In addition, the `types.StargateQuerier` references a global `types.QueryRouter` which points
to `baseapp.GRPCQueryRouter`.

This design is based on wasmd's (v0.50.0) querier plugin design.
*/

var _ wasmvmtypes.Querier = (*queryHandler)(nil)

// defaultAcceptList defines a set of default allowed queries made available to the Querier.
var defaultAcceptList = []string{
	"/ibc.core.client.v1.Query/VerifyMembership",
}

// queryHandler is a wrapper around the sdk Environment and the CallerID that calls
// into the query plugins.
type queryHandler struct {
	appmodule.Environment
	Ctx      context.Context
	Plugins  QueryPlugins
	CallerID string
}

// newQueryHandler returns a default querier that can be used in the contract.
func newQueryHandler(ctx context.Context, env appmodule.Environment, plugins QueryPlugins, callerID string) *queryHandler {
	return &queryHandler{
		Ctx:         ctx,
		Environment: env,
		Plugins:     plugins,
		CallerID:    callerID,
	}
}

// GasConsumed implements the wasmvmtypes.Querier interface.
func (q *queryHandler) GasConsumed() uint64 {
	return VMGasRegister.ToWasmVMGas(q.GasService.GasMeter(q.Ctx).Consumed())
}

// Query implements the wasmvmtypes.Querier interface.
func (q *queryHandler) Query(request wasmvmtypes.QueryRequest, gasLimit uint64) ([]byte, error) {
	sdkGasLimit := VMGasRegister.FromWasmVMGas(gasLimit)
	// discard all changes/events in subCtx by not committing the cached context
	var res []byte
	_, err := q.BranchService.ExecuteWithGasLimit(q.Ctx, sdkGasLimit, func(ctx context.Context) error {
		var err error
		res, err = q.Plugins.HandleQuery(ctx, q.CallerID, request)
		if err == nil {
			return nil
		}

		q.Logger.Debug("Redacting query error", "cause", err)
		return err
	})
	if err != nil {
		return nil, redactError(err)
	}
	return res, nil
}

type (
	CustomQuerier   func(ctx context.Context, request json.RawMessage) ([]byte, error)
	StargateQuerier func(ctx context.Context, request *wasmvmtypes.StargateQuery) ([]byte, error)

	// QueryPlugins is a list of queriers that can be used to extend the default querier.
	QueryPlugins struct {
		Custom   CustomQuerier
		Stargate StargateQuerier
	}
)

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

// HandleQuery implements the ibcwasm.QueryPluginsI interface.
func (e QueryPlugins) HandleQuery(ctx context.Context, caller string, request wasmvmtypes.QueryRequest) ([]byte, error) {
	if request.Stargate != nil {
		return e.Stargate(ctx, request.Stargate)
	}

	if request.Custom != nil {
		return e.Custom(ctx, request.Custom)
	}

	return nil, wasmvmtypes.UnsupportedRequest{Kind: "Unsupported query request"}
}

// NewDefaultQueryPlugins returns the default set of query plugins
func NewDefaultQueryPlugins(queryRouter types.QueryRouter) QueryPlugins {
	return QueryPlugins{
		Custom:   RejectCustomQuerier(),
		Stargate: AcceptListStargateQuerier([]string{}, queryRouter),
	}
}

// AcceptListStargateQuerier allows all queries that are in the provided accept list.
// This function returns protobuf encoded responses in bytes.
func AcceptListStargateQuerier(acceptedQueries []string, queryRouter types.QueryRouter) func(context.Context, *wasmvmtypes.StargateQuery) ([]byte, error) {
	return func(ctx context.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
		// append user defined accepted queries to default list defined above.
		acceptedQueries = append(defaultAcceptList, acceptedQueries...)

		isAccepted := slices.Contains(acceptedQueries, request.Path)
		if !isAccepted {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", request.Path)}
		}

		route := queryRouter.Route(request.Path)
		if route == nil {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("No route to query '%s'", request.Path)}
		}

		res, err := route(sdk.UnwrapSDKContext(ctx), &abci.QueryRequest{
			Data: request.Data,
			Path: request.Path,
		})
		if err != nil {
			return nil, err
		}
		if res == nil || res.Value == nil {
			return nil, wasmvmtypes.InvalidResponse{Err: "Query response is empty"}
		}

		return res.Value, nil
	}
}

// RejectCustomQuerier rejects all custom queries
func RejectCustomQuerier() func(context.Context, json.RawMessage) ([]byte, error) {
	return func(ctx context.Context, request json.RawMessage) ([]byte, error) {
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
