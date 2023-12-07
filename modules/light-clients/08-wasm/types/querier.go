package types

import (
	"encoding/json"
	"errors"
	"fmt"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"

	abci "github.com/cometbft/cometbft/abci/types"
)

var _ wasmvmtypes.Querier = (*DefaultQuerier)(nil)

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

type QueryPlugins struct {
	// Bank         func(ctx sdk.Context, request *wasmvmtypes.BankQuery) ([]byte, error)
	Custom func(ctx sdk.Context, request json.RawMessage) ([]byte, error)
	// IBC          func(ctx sdk.Context, caller sdk.AccAddress, request *wasmvmtypes.IBCQuery) ([]byte, error)
	// Staking      func(ctx sdk.Context, request *wasmvmtypes.StakingQuery) ([]byte, error)
	Stargate func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error)
	// Wasm         func(ctx sdk.Context, request *wasmvmtypes.WasmQuery) ([]byte, error)
	// Distribution func(ctx sdk.Context, request *wasmvmtypes.DistributionQuery) ([]byte, error)
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
		return handleStargateQuery(subCtx, request.Stargate)
	}

	return nil, wasmvmtypes.UnsupportedRequest{Kind: "non-stargate queries in contract are not allowed"}
}

var acceptList = map[string]struct{}{
	"/ibc.core.client.v1.Query/VerifyMembershipProof": {},
}

// handleStargateQuery supports a preconfigured set of stargate queries only.
// All arguments must be non nil.
//
// Warning: Chains need to test and maintain their accept list carefully.
// There were critical consensus breaking issues in the past with non-deterministic behavior in the SDK.
func handleStargateQuery(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
	_, accepted := acceptList[request.Path]
	if !accepted {
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
		return nil, errors.New("empty response value")
	}

	return res.Value, nil
}
