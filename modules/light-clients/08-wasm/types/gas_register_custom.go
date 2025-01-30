package types

import (
	"fmt"
	"math"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	"cosmossdk.io/core/gas"
	storetypes "cosmossdk.io/store/types"
)

// While gas_register.go is a direct copy of https://github.com/CosmWasm/wasmd/blob/main/x/wasm/types/gas_register.go
// This file contains additional constructs that can be maintained separately.
// Most of these functions are slight modifications of keeper function from wasmd, which act on `WasmGasRegister` instead of `Keeper`.
const (
	// DefaultDeserializationCostPerByte The formula should be `len(data) * deserializationCostPerByte`
	DefaultDeserializationCostPerByte = 1
)

var CostJSONDeserialization = wasmvmtypes.UFraction{
	Numerator:   DefaultDeserializationCostPerByte * DefaultGasMultiplier,
	Denominator: 1,
}

func (g WasmGasRegister) RuntimeGasForContract(meter gas.Meter) uint64 {
	if meter.Remaining() >= meter.Limit() {
		return 0
	}
	// infinite gas meter with limit=0 or MaxUint64
	if meter.Limit() == 0 || meter.Limit() == math.MaxUint64 {
		return math.MaxUint64
	}
	var consumedToLimit gas.Gas
	if meter.Remaining() <= meter.Limit() {
		consumedToLimit = meter.Limit()
	} else {
		consumedToLimit = meter.Consumed()
	}
	return g.ToWasmVMGas(meter.Limit() - consumedToLimit)
}

func (g WasmGasRegister) ConsumeRuntimeGas(meter gas.Meter, gasAmt uint64) {
	consumed := g.FromWasmVMGas(gasAmt)
	if err := meter.Consume(consumed, "wasm contract"); err != nil {
		panic(fmt.Errorf("ConsumeRuntimeGas: failed to consume gas: %w", err))
	}

	if meter.Consumed() >= meter.Limit() {
		// throw OutOfGas error if we ran out (got exactly to zero due to better limit enforcing)
		panic(storetypes.ErrorOutOfGas{Descriptor: "Wasmer function execution"})
	}
}

// MultipliedGasMeter wraps the GasMeter from context and multiplies all reads by out defined multiplier
type MultipliedGasMeter struct {
	originalMeter gas.Meter
	GasRegister   GasRegister
}

func NewMultipliedGasMeter(originalMeter gas.Meter, gr GasRegister) MultipliedGasMeter {
	return MultipliedGasMeter{originalMeter: originalMeter, GasRegister: gr}
}

var _ wasmvm.GasMeter = MultipliedGasMeter{}

func (m MultipliedGasMeter) GasConsumed() storetypes.Gas {
	return m.GasRegister.ToWasmVMGas(m.originalMeter.Consumed())
}
