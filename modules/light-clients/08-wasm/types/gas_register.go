package types

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	// DefaultGasMultiplier is how many CosmWasm gas points = 1 Cosmos SDK gas point.
	//
	// CosmWasm gas strategy is documented in https://github.com/CosmWasm/cosmwasm/blob/v1.0.0-beta/docs/GAS.md.
	// Cosmos SDK reference costs can be found here: https://github.com/cosmos/cosmos-sdk/blob/v0.42.10/store/types/gas.go#L198-L209.
	//
	// The original multiplier of 100 up to CosmWasm 0.16 was based on
	//     "A write at ~3000 gas and ~200us = 10 gas per us (microsecond) cpu/io
	//     Rough timing have 88k gas at 90us, which is equal to 1k sdk gas... (one read)"
	// as well as manual Wasmer benchmarks from 2019. This was then multiplied by 150_000
	// in the 0.16 -> 1.0 upgrade (https://github.com/CosmWasm/cosmwasm/pull/1120).
	// In the 2.0 upgrade, this was reduced by a factor of 1000 (https://github.com/CosmWasm/cosmwasm/pull/1884).
	//
	// The multiplier deserves more reproducible benchmarking and a strategy that allows easy adjustments.
	// This is tracked in https://github.com/CosmWasm/wasmd/issues/566 and https://github.com/CosmWasm/wasmd/issues/631.
	// Gas adjustments are consensus breaking but may happen in any release marked as consensus breaking.
	// Do not make assumptions on how much gas an operation will consume in places that are hard to adjust,
	// such as hardcoding them in contracts.
	//
	// Please note that all gas prices returned to wasmvm should have this multiplied.
	// Benchmarks and numbers were discussed in: https://github.com/CosmWasm/wasmd/pull/634#issuecomment-938055852
	DefaultGasMultiplier uint64 = 140_000
	// DefaultInstanceCost is how much SDK gas we charge each time we load a WASM instance.
	// Creating a new instance is costly, and this helps put a recursion limit to contracts calling contracts.
	// Benchmarks and numbers were discussed in: https://github.com/CosmWasm/wasmd/pull/634#issuecomment-938056803
	DefaultInstanceCost uint64 = 60_000
	// DefaultInstanceCostDiscount is charged instead of DefaultInstanceCost for cases where
	// we assume the contract is loaded from an in-memory cache.
	// For a long time it was implicitly just 0 in those cases.
	// Now we use something small that roughly reflects the 45µs startup time (30x cheaper than DefaultInstanceCost).
	DefaultInstanceCostDiscount uint64 = 2_000
	// DefaultCompileCost is how much SDK gas is charged *per byte* for compiling WASM code.
	// Benchmarks and numbers were discussed in: https://github.com/CosmWasm/wasmd/pull/634#issuecomment-938056803
	DefaultCompileCost uint64 = 3
	// DefaultEventAttributeDataCost is how much SDK gas is charged *per byte* for attribute data in events.
	// This is used with len(key) + len(value)
	DefaultEventAttributeDataCost uint64 = 1
	// DefaultContractMessageDataCost is how much SDK gas is charged *per byte* of the message that goes to the contract
	// This is used with len(msg). Note that the message is deserialized in the receiving contract and this is charged
	// with wasm gas already. The derserialization of results is also charged in wasmvm. I am unsure if we need to add
	// additional costs here.
	// Note: also used for error fields on reply, and data on reply. Maybe these should be pulled out to a different (non-zero) field
	DefaultContractMessageDataCost uint64 = 0
	// DefaultPerAttributeCost is how much SDK gas we charge per attribute count.
	DefaultPerAttributeCost uint64 = 10
	// DefaultPerCustomEventCost is how much SDK gas we charge per event count.
	DefaultPerCustomEventCost uint64 = 20
	// DefaultEventAttributeDataFreeTier number of bytes of total attribute data we do not charge.
	DefaultEventAttributeDataFreeTier = 100
)

// default: 0.15 gas.
// see https://github.com/CosmWasm/wasmd/pull/898#discussion_r937727200
var (
	defaultPerByteUncompressCost = wasmvmtypes.UFraction{
		Numerator:   15,
		Denominator: 100,
	}

	VMGasRegister = NewDefaultWasmGasRegister()
)

// DefaultPerByteUncompressCost is how much SDK gas we charge per source byte to unpack
func DefaultPerByteUncompressCost() wasmvmtypes.UFraction {
	return defaultPerByteUncompressCost
}

// GasRegister abstract source for gas costs
type GasRegister interface {
	// UncompressCosts costs to unpack a new wasm contract
	UncompressCosts(byteLength int) storetypes.Gas
	// SetupContractCost are charged when interacting with a Wasm contract, i.e. every time
	// the contract is prepared for execution through any entry point (execute/instantiate/sudo/query/ibc_*/...).
	SetupContractCost(discount bool, msgLen int) storetypes.Gas
	// ReplyCosts costs to handle a message reply
	ReplyCosts(discount bool, reply wasmvmtypes.Reply) storetypes.Gas
	// EventCosts costs to persist an event
	EventCosts(attrs []wasmvmtypes.EventAttribute, events wasmvmtypes.Array[wasmvmtypes.Event]) storetypes.Gas
	// ToWasmVMGas converts from Cosmos SDK gas units to [CosmWasm gas] (aka. wasmvm gas)
	//
	// [CosmWasm gas]: https://github.com/CosmWasm/cosmwasm/blob/v1.3.1/docs/GAS.md
	ToWasmVMGas(source storetypes.Gas) uint64
	// FromWasmVMGas converts from [CosmWasm gas] (aka. wasmvm gas) to Cosmos SDK gas units
	//
	// [CosmWasm gas]: https://github.com/CosmWasm/cosmwasm/blob/v1.3.1/docs/GAS.md
	FromWasmVMGas(source uint64) storetypes.Gas
}

// WasmGasRegisterConfig config type
type WasmGasRegisterConfig struct {
	// InstanceCost are charged when interacting with a Wasm contract.
	// "Instance" refers to the in-memory Instance of the Wasm runtime, not the contract address on chain.
	// InstanceCost are part of a contract's setup cost.
	InstanceCost storetypes.Gas
	// InstanceCostDiscount is a discounted version of InstanceCost. It is charged whenever
	// we can reasonably assume that a contract is in one of the in-memory caches. E.g.
	// when the contract is pinned or we send a reply to a contract that was executed before.
	// See also https://github.com/CosmWasm/wasmd/issues/1798 for more thinking around
	// discount cases.
	InstanceCostDiscount storetypes.Gas
	// CompileCost costs to persist and "compile" a new wasm contract
	CompileCost storetypes.Gas
	// UncompressCost costs per byte to unpack a contract
	UncompressCost wasmvmtypes.UFraction
	// GasMultiplier is how many cosmwasm gas points = 1 sdk gas point
	// SDK reference costs can be found here: https://github.com/cosmos/cosmos-sdk/blob/02c6c9fafd58da88550ab4d7d494724a477c8a68/store/types/gas.go#L153-L164
	GasMultiplier storetypes.Gas
	// EventPerAttributeCost is how much SDK gas is charged *per byte* for attribute data in events.
	// This is used with len(key) + len(value)
	EventPerAttributeCost storetypes.Gas
	// EventAttributeDataCost is how much SDK gas is charged *per byte* for attribute data in events.
	// This is used with len(key) + len(value)
	EventAttributeDataCost storetypes.Gas
	// EventAttributeDataFreeTier number of bytes of total attribute data that is free of charge
	EventAttributeDataFreeTier uint64
	// ContractMessageDataCost SDK gas charged *per byte* of the message that goes to the contract
	// This is used with len(msg)
	ContractMessageDataCost storetypes.Gas
	// CustomEventCost cost per custom event
	CustomEventCost uint64
}

// DefaultGasRegisterConfig default values
func DefaultGasRegisterConfig() WasmGasRegisterConfig {
	return WasmGasRegisterConfig{
		InstanceCost:               DefaultInstanceCost,
		InstanceCostDiscount:       DefaultInstanceCostDiscount,
		CompileCost:                DefaultCompileCost,
		GasMultiplier:              DefaultGasMultiplier,
		EventPerAttributeCost:      DefaultPerAttributeCost,
		CustomEventCost:            DefaultPerCustomEventCost,
		EventAttributeDataCost:     DefaultEventAttributeDataCost,
		EventAttributeDataFreeTier: DefaultEventAttributeDataFreeTier,
		ContractMessageDataCost:    DefaultContractMessageDataCost,
		UncompressCost:             DefaultPerByteUncompressCost(),
	}
}

// WasmGasRegister implements GasRegister interface
type WasmGasRegister struct {
	c WasmGasRegisterConfig
}

// NewDefaultWasmGasRegister creates instance with default values
func NewDefaultWasmGasRegister() WasmGasRegister {
	return NewWasmGasRegister(DefaultGasRegisterConfig())
}

// NewWasmGasRegister constructor
func NewWasmGasRegister(c WasmGasRegisterConfig) WasmGasRegister {
	if c.GasMultiplier == 0 {
		panic(errorsmod.Wrap(sdkerrors.ErrLogic, "GasMultiplier can not be 0"))
	}
	return WasmGasRegister{
		c: c,
	}
}

// UncompressCosts costs to unpack a new wasm contract
func (g WasmGasRegister) UncompressCosts(byteLength int) storetypes.Gas {
	if byteLength < 0 {
		panic(errorsmod.Wrap(ErrInvalid, "negative length"))
	}
	return g.c.UncompressCost.Mul(uint64(byteLength)).Floor()
}

// SetupContractCost costs when interacting with a wasm contract.
// Set discount to true in cases where you can reasonably assume the contract
// is loaded from an in-memory cache (e.g. pinned contracts or replies).
func (g WasmGasRegister) SetupContractCost(discount bool, msgLen int) storetypes.Gas {
	if msgLen < 0 {
		panic(errorsmod.Wrap(ErrInvalid, "negative length"))
	}
	dataCost := storetypes.Gas(msgLen) * g.c.ContractMessageDataCost
	if discount {
		return g.c.InstanceCostDiscount + dataCost
	}
	return g.c.InstanceCost + dataCost
}

// ReplyCosts costs to handle a message reply.
// Set discount to true in cases where you can reasonably assume the contract
// is loaded from an in-memory cache (e.g. pinned contracts or replies).
func (g WasmGasRegister) ReplyCosts(discount bool, reply wasmvmtypes.Reply) storetypes.Gas {
	var eventGas storetypes.Gas
	msgLen := len(reply.Result.Err)
	if reply.Result.Ok != nil {
		msgLen += len(reply.Result.Ok.Data)
		var attrs []wasmvmtypes.EventAttribute
		for _, e := range reply.Result.Ok.Events {
			eventGas += storetypes.Gas(len(e.Type)) * g.c.EventAttributeDataCost
			attrs = append(attrs, e.Attributes...)
		}
		// apply free tier on the whole set not per event
		eventGas += g.EventCosts(attrs, nil)
	}
	return eventGas + g.SetupContractCost(discount, msgLen)
}

// EventCosts costs to persist an event
func (g WasmGasRegister) EventCosts(attrs []wasmvmtypes.EventAttribute, events wasmvmtypes.Array[wasmvmtypes.Event]) storetypes.Gas {
	gas, remainingFreeTier := g.eventAttributeCosts(attrs, g.c.EventAttributeDataFreeTier)
	for _, e := range events {
		gas += g.c.CustomEventCost
		gas += storetypes.Gas(len(e.Type)) * g.c.EventAttributeDataCost // no free tier with event type
		var attrCost storetypes.Gas
		attrCost, remainingFreeTier = g.eventAttributeCosts(e.Attributes, remainingFreeTier)
		gas += attrCost
	}
	return gas
}

func (g WasmGasRegister) eventAttributeCosts(attrs []wasmvmtypes.EventAttribute, freeTier uint64) (storetypes.Gas, uint64) {
	if len(attrs) == 0 {
		return 0, freeTier
	}
	var storedBytes uint64
	for _, l := range attrs {
		storedBytes += uint64(len(l.Key)) + uint64(len(l.Value))
	}
	storedBytes, freeTier = calcWithFreeTier(storedBytes, freeTier)
	// total Length * costs + attribute count * costs
	r := sdkmath.NewIntFromUint64(g.c.EventAttributeDataCost).Mul(sdkmath.NewIntFromUint64(storedBytes)).
		Add(sdkmath.NewIntFromUint64(g.c.EventPerAttributeCost).Mul(sdkmath.NewIntFromUint64(uint64(len(attrs)))))
	if !r.IsUint64() {
		panic(storetypes.ErrorOutOfGas{Descriptor: "overflow"})
	}
	return r.Uint64(), freeTier
}

// apply free tier
func calcWithFreeTier(storedBytes, freeTier uint64) (uint64, uint64) {
	if storedBytes <= freeTier {
		return 0, freeTier - storedBytes
	}
	storedBytes -= freeTier
	return storedBytes, 0
}

// ToWasmVMGas converts from Cosmos SDK gas units to [CosmWasm gas] (aka. wasmvm gas)
//
// [CosmWasm gas]: https://github.com/CosmWasm/cosmwasm/blob/v1.3.1/docs/GAS.md
func (g WasmGasRegister) ToWasmVMGas(source storetypes.Gas) uint64 {
	x := source * g.c.GasMultiplier
	if x < source {
		panic(storetypes.ErrorOutOfGas{Descriptor: "overflow"})
	}
	return x
}

// FromWasmVMGas converts from [CosmWasm gas] (aka. wasmvm gas) to Cosmos SDK gas units
//
// [CosmWasm gas]: https://github.com/CosmWasm/cosmwasm/blob/v1.3.1/docs/GAS.md
func (g WasmGasRegister) FromWasmVMGas(source uint64) storetypes.Gas {
	return source / g.c.GasMultiplier
}
