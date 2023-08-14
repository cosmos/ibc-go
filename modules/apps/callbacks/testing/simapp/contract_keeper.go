package simapp

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	callbacktypes "github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

// MockKeeper implements callbacktypes.ContractKeeper
var _ callbacktypes.ContractKeeper = (*ContractKeeper)(nil)

var (
	StatefulCounterKey              = "stateful-callback-counter"
	MockCallbackUnauthorizedAddress = "cosmos15ulrf36d4wdtrtqzkgaan9ylwuhs7k7qz753uk"
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

// IBCPacketSendCallback returns nil if the gas meter has greater than
// or equal to 500_000 gas remaining.
// This function oog panics if the gas remaining is less than 500_000.
// This function errors if the authAddress is MockCallbackUnauthorizedAddress.
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
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeSendPacket, packetSenderAddress)
}

// IBCOnAcknowledgementPacketCallback returns nil if the gas meter has greater than
// or equal to 500_000 gas remaining.
// This function oog panics if the gas remaining is less than 500_000.
// This function errors if the authAddress is MockCallbackUnauthorizedAddress.
func (k ContractKeeper) IBCOnAcknowledgementPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeAcknowledgementPacket, packetSenderAddress)
}

// IBCOnTimeoutPacketCallback returns nil if the gas meter has greater than
// or equal to 500_000 gas remaining.
// This function oog panics if the gas remaining is less than 500_000.
// This function errors if the authAddress is MockCallbackUnauthorizedAddress.
func (k ContractKeeper) IBCOnTimeoutPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeTimeoutPacket, packetSenderAddress)
}

// IBCReceivePacketCallback returns nil if the gas meter has greater than
// or equal to 500_000 gas remaining.
// This function oog panics if the gas remaining is less than 500_000.
// This function errors if the authAddress is MockCallbackUnauthorizedAddress.
func (k ContractKeeper) IBCReceivePacketCallback(
	ctx sdk.Context,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
	contractAddress string,
) error {
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeReceivePacket, "")
}

// processMockCallback returns nil if the gas meter has greater than or equal to 500_000 gas remaining.
// This function oog panics if the gas remaining is less than 500_000.
// This function errors if the authAddress is MockCallbackUnauthorizedAddress.
func (k ContractKeeper) processMockCallback(
	ctx sdk.Context,
	callbackType callbacktypes.CallbackType,
	authAddress string,
) error {
	gasRemaining := ctx.GasMeter().GasRemaining()

	// increment stateful entries, if the callbacks module handler
	// reverts state, we can check by querying for the counter
	// currently stored.
	k.IncrementStateEntryCounter(ctx)

	// increment callback execution attempts
	k.Counters[callbackType]++

	if gasRemaining < 500000 {
		// consume gas will panic since we attempt to consume 500_000 gas, for tests
		ctx.GasMeter().ConsumeGas(500000, fmt.Sprintf("mock %s callback panic", callbackType))
	}

	if authAddress == MockCallbackUnauthorizedAddress {
		ctx.GasMeter().ConsumeGas(500000, fmt.Sprintf("mock %s callback unauthorized", callbackType))
		return ibcmock.MockApplicationCallbackError
	}

	ctx.GasMeter().ConsumeGas(500000, fmt.Sprintf("mock %s callback success", callbackType))
	return nil
}
