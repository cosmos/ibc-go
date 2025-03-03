package internal

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/types"
)

// ProcessCallback executes the callbackExecutor and reverts contract changes if the callbackExecutor fails.
//
// Error Precedence and Returns:
//   - oogErr: Takes the highest precedence. If the callback runs out of gas, an error wrapped with types.ErrCallbackOutOfGas is returned.
//   - panicErr: Takes the second-highest precedence. If a panic occurs and it is not propagated, an error wrapped with types.ErrCallbackPanic is returned.
//   - callbackErr: If the callbackExecutor returns an error, it is returned as-is.
//
// panics if
//   - the contractExecutor panics for any reason, and the callbackType is SendPacket, or
//   - the contractExecutor runs out of gas and the relayer has not reserved gas grater than or equal to
//     CommitGasLimit.
func ProcessCallback(
	ctx sdk.Context, callbackType types.CallbackType,
	callbackData types.CallbackData, callbackExecutor func(sdk.Context) error,
) (err error) {
	cachedCtx, writeFn := ctx.CacheContext()
	cachedCtx = cachedCtx.WithGasMeter(storetypes.NewGasMeter(callbackData.ExecutionGasLimit))

	defer func() {
		// consume the minimum of g.consumed and g.limit
		ctx.GasMeter().ConsumeGas(cachedCtx.GasMeter().GasConsumedToLimit(), fmt.Sprintf("ibc %s callback", callbackType))

		// recover from all panics except during SendPacket callbacks
		if r := recover(); r != nil {
			if callbackType == types.CallbackTypeSendPacket {
				panic(r)
			}
			err = errorsmod.Wrapf(types.ErrCallbackPanic, "ibc %s callback panicked with: %v", callbackType, r)
		}

		// if the callback ran out of gas and the relayer has not reserved enough gas, then revert the state
		if cachedCtx.GasMeter().IsPastLimit() {
			if callbackData.AllowRetry() {
				panic(storetypes.ErrorOutOfGas{Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", callbackType, callbackData.CommitGasLimit)})
			}
			err = errorsmod.Wrapf(types.ErrCallbackOutOfGas, "ibc %s callback out of gas", callbackType)
		}

		// allow the transaction to be committed, continuing the packet lifecycle
	}()

	err = callbackExecutor(cachedCtx)
	if err == nil {
		writeFn()
	}

	return err
}
