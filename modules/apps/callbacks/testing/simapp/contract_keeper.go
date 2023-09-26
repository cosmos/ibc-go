package simapp

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	callbacktypes "github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
)

// MockKeeper implements callbacktypes.ContractKeeper
var _ callbacktypes.ContractKeeper = (*ContractKeeper)(nil)

var StatefulCounterKey = "stateful-callback-counter"

const (
	// OogPanicContract is a contract address that will panic out of gas
	OogPanicContract = "panics out of gas"
	// OogErrorContract is a contract address that will error out of gas
	OogErrorContract = "errors out of gas"
	// PanicContract is a contract address that will panic
	PanicContract = "panics"
	// ErrorContract is a contract address that will return an error
	ErrorContract = "errors"
	// SuccessContract is a contract address that will return nil
	SuccessContract = "success"
)

// This is a mock contract keeper used for testing. It is not wired up to any modules.
// It implements the interface functions expected by the ibccallbacks middleware
// so that it can be tested with simapp. The keeper is responsible for tracking
// two metrics:
//   - number of callbacks called per callback type
//   - stateful entry attempts
//
// The counter for callbacks allows us to ensure the correct callbacks were routed to
// and the stateful entries allows us to track state reversals or reverted state upon
// contract execution failure or out of gas errors.
type ContractKeeper struct {
	key storetypes.StoreKey

	Counters map[callbacktypes.CallbackType]int
}

// SetStateEntryCounter sets state entry counter. The number of stateful
// entries is tracked as a uint8. This function is used to test state reversals.
func (k ContractKeeper) SetStateEntryCounter(ctx sdk.Context, count uint8) {
	store := ctx.KVStore(k.key)
	store.Set([]byte(StatefulCounterKey), []byte{count})
}

// GetStateEntryCounter returns the state entry counter stored in state.
func (k ContractKeeper) GetStateEntryCounter(ctx sdk.Context) uint8 {
	store := ctx.KVStore(k.key)
	bz := store.Get([]byte(StatefulCounterKey))
	if bz == nil {
		return 0
	}
	return bz[0]
}

// IncrementStatefulCounter increments the stateful callback counter in state.
func (k ContractKeeper) IncrementStateEntryCounter(ctx sdk.Context) {
	count := k.GetStateEntryCounter(ctx)
	k.SetStateEntryCounter(ctx, count+1)
}

// NewKeeper creates a new mock ContractKeeper.
func NewContractKeeper(key storetypes.StoreKey) ContractKeeper {
	return ContractKeeper{
		key:      key,
		Counters: make(map[callbacktypes.CallbackType]int),
	}
}

// IBCPacketSendCallback increments the stateful entry counter and the send_packet callback counter.
// This function:
//   - returns MockApplicationCallbackError and consumes half the remaining gas if the contract address is ErrorContract
//   - Oog panics and consumes all the remaining gas + 1 if the contract address is OogPanicContract
//   - returns MockApplicationCallbackError and consumes all the remaining gas + 1 if the contract address is OogErrorContract
//   - Panics and consumes half the remaining gas if the contract address is PanicContract
//   - returns nil and consumes half the remaining gas if the contract address is SuccessContract or any other value
func (k ContractKeeper) IBCSendPacketCallback(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	packetData []byte,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeSendPacket, contractAddress)
}

// IBCOnAcknowledgementPacketCallback increments the stateful entry counter and the acknowledgement_packet callback counter.
// This function:
//   - returns MockApplicationCallbackError and consumes half the remaining gas if the contract address is ErrorContract
//   - Oog panics and consumes all the remaining gas + 1 if the contract address is OogPanicContract
//   - returns MockApplicationCallbackError and consumes all the remaining gas + 1 if the contract address is OogErrorContract
//   - Panics and consumes half the remaining gas if the contract address is PanicContract
//   - returns nil and consumes half the remaining gas if the contract address is SuccessContract or any other value
func (k ContractKeeper) IBCOnAcknowledgementPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeAcknowledgementPacket, contractAddress)
}

// IBCOnTimeoutPacketCallback increments the stateful entry counter and the timeout_packet callback counter.
// This function:
//   - returns MockApplicationCallbackError and consumes half the remaining gas if the contract address is ErrorContract
//   - Oog panics and consumes all the remaining gas + 1 if the contract address is OogPanicContract
//   - returns MockApplicationCallbackError and consumes all the remaining gas + 1 if the contract address is OogErrorContract
//   - Panics and consumes half the remaining gas if the contract address is PanicContract
//   - returns nil and consumes half the remaining gas if the contract address is SuccessContract or any other value
func (k ContractKeeper) IBCOnTimeoutPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeTimeoutPacket, contractAddress)
}

// IBCReceivePacketCallback increments the stateful entry counter and the receive_packet callback counter.
// This function:
//   - returns MockApplicationCallbackError and consumes half the remaining gas if the contract address is ErrorContract
//   - Oog panics and consumes all the remaining gas + 1 if the contract address is OogPanicContract
//   - returns MockApplicationCallbackError and consumes all the remaining gas + 1 if the contract address is OogErrorContract
//   - Panics and consumes half the remaining gas if the contract address is PanicContract
//   - returns nil and consumes half the remaining gas if the contract address is SuccessContract or any other value
func (k ContractKeeper) IBCReceivePacketCallback(
	ctx sdk.Context,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
	contractAddress string,
) error {
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeReceivePacket, contractAddress)
}

// processMockCallback processes a mock callback.
// It increments the stateful entry counter and the callback counter.
// This function:
//   - returns MockApplicationCallbackError and consumes half the remaining gas if the contract address is ErrorContract
//   - Oog panics and consumes all the remaining gas + 1 if the contract address is OogPanicContract
//   - returns MockApplicationCallbackError and consumes all the remaining gas + 1 if the contract address is OogErrorContract
//   - Panics and consumes half the remaining gas if the contract address is PanicContract
//   - returns nil and consumes half the remaining gas if the contract address is SuccessContract or any other value
func (k ContractKeeper) processMockCallback(
	ctx sdk.Context,
	callbackType callbacktypes.CallbackType,
	contractAddress string,
) (err error) {
	gasRemaining := ctx.GasMeter().GasRemaining()

	// increment stateful entries, if the callbacks module handler
	// reverts state, we can check by querying for the counter
	// currently stored.
	k.IncrementStateEntryCounter(ctx)

	// increment callback execution attempts
	k.Counters[callbackType]++

	switch contractAddress {
	case ErrorContract:
		// consume half of the remaining gas so that ConsumeGas cannot oog panic
		ctx.GasMeter().ConsumeGas(gasRemaining/2, fmt.Sprintf("mock %s callback unauthorized", callbackType))
		return ibcmock.MockApplicationCallbackError
	case OogPanicContract:
		ctx.GasMeter().ConsumeGas(gasRemaining+1, fmt.Sprintf("mock %s callback oog panic", callbackType))
		return nil // unreachable
	case OogErrorContract:
		defer func() {
			_ = recover()
			err = ibcmock.MockApplicationCallbackError
		}()
		ctx.GasMeter().ConsumeGas(gasRemaining+1, fmt.Sprintf("mock %s callback oog error", callbackType))
		return nil // unreachable
	case PanicContract:
		// consume half of the remaining gas so that ConsumeGas cannot oog panic
		ctx.GasMeter().ConsumeGas(gasRemaining/2, fmt.Sprintf("mock %s callback panic", callbackType))
		panic(ibcmock.MockApplicationCallbackError)
	default:
		// consume half of the remaining gas so that ConsumeGas cannot oog panic
		ctx.GasMeter().ConsumeGas(gasRemaining/2, fmt.Sprintf("mock %s callback success", callbackType))
		return nil // success
	}
}
