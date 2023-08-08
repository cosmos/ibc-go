package mock

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	callbacktypes "github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// MockKeeper implements callbacktypes.ContractKeeper
var _ callbacktypes.ContractKeeper = (*Keeper)(nil)

// Keeper can be used to mock the expected keepers needed for testing.
//
// Keeper currently mocks the following interfaces:
//   - callbacktypes.ContractKeeper
type Keeper struct {
	ContractKeeper

	key storetypes.StoreKey
}

// This is a mock keeper used for testing. It is not wired up to any modules.
// It implements the interface functions expected by the ibccallbacks middleware
// so that it can be tested with simapp. The keeper is responsible for tracking
// two metrics:
// - number of callbacks called per callback type
// - stateful entry attempts
//
// The counter for callbacks allows us to ensure the correct callbacks were routed to
// and the stateful entries allows us to track state reversals or reverted state upon
// contract execution failure or out of gas errors.
type ContractKeeper struct {
	Counters map[callbacktypes.CallbackType]int
}

// SetStateCounter sets the stateful callback counter in state.
// This function is used to test state reversals. The callback counters
// directly listed under MockContractKeeper will not be reversed if the
// state is reversed.
func (k Keeper) SetStateCounter(ctx sdk.Context, count uint8) {
	store := ctx.KVStore(k.key)
	store.Set([]byte(StatefulCounterKey), []byte{count})
}

// GetStateCounter returns the stateful callback counter from state.
func (k Keeper) GetStateCounter(ctx sdk.Context) uint8 {
	store := ctx.KVStore(k.key)
	bz := store.Get([]byte(StatefulCounterKey))
	if bz == nil {
		return 0
	}
	return bz[0]
}

// IncrementStatefulCounter increments the stateful callback counter in state.
func (k Keeper) IncrementStatefulCounter(ctx sdk.Context) {
	count := k.GetStateCounter(ctx)
	k.SetStateCounter(ctx, count+1)
}

// NewKeeper creates a new mock Keeper.
func NewMockKeeper(key storetypes.StoreKey) Keeper {
	return Keeper{
		key: key,
		ContractKeeper: ContractKeeper{
			Counters: make(map[callbacktypes.CallbackType]int)},
	}
}

// IBCPacketSendCallback returns nil if the gas meter has greater than
// or equal to 500000 gas remaining.
// This function consumes 500000 gas, or the remaining gas if less than 500000.
// This function oog panics if the gas remaining is less than 400000.
func (k Keeper) IBCSendPacketCallback(
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
// or equal to 500000 gas remaining.
// This function consumes 500000 gas, or the remaining gas if less than 500000.
// This function oog panics if the gas remaining is less than 400000.
func (k Keeper) IBCOnAcknowledgementPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeAcknowledgement, packetSenderAddress)
}

// IBCOnTimeoutPacketCallback returns nil if the gas meter has greater than
// or equal to 500000 gas remaining.
// This function consumes 500000 gas, or the remaining gas if less than 500000.
// This function oog panics if the gas remaining is less than 400000.
func (k Keeper) IBCOnTimeoutPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeTimeoutPacket, packetSenderAddress)
}

// IBCWriteAcknowledgementCallback returns nil if the gas meter has greater than
// or equal to 500000 gas remaining.
// This function consumes 500000 gas, or the remaining gas if less than 500000.
// This function oog panics if the gas remaining is less than 400000.
func (k Keeper) IBCWriteAcknowledgementCallback(
	ctx sdk.Context,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
	contractAddress string,
) error {
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeWriteAcknowledgement, "")
}

// processMockCallback returns nil if the gas meter has greater than or equal to 500000 gas remaining.
// This function consumes 500000 gas, or the remaining gas if less than 500000.
// This function oog panics if the gas remaining is less than 400000.
func (k Keeper) processMockCallback(
	ctx sdk.Context,
	callbackType callbacktypes.CallbackType,
	authAddress string,
) error {
	gasRemaining := ctx.GasMeter().GasRemaining()

	// increment stateful entries, if the callbacks module handler
	// reverts state, we can check by querying for the counter
	// currently stored.
	k.IncrementStatefulCounter(ctx)

	// increment callback execution attempts
	k.Counters[callbackType]++

	if gasRemaining < 400000 {
		// consume gas will panic since we attempt to consume 500_000 gas, for tests
		ctx.GasMeter().ConsumeGas(500000, fmt.Sprintf("mock %s callback panic", callbackType))
	} else if gasRemaining < 500000 {
		ctx.GasMeter().ConsumeGas(gasRemaining, fmt.Sprintf("mock %s callback failure", callbackType))
		return MockApplicationCallbackError
	}

	if authAddress == MockCallbackUnauthorizedAddress {
		ctx.GasMeter().ConsumeGas(500000, fmt.Sprintf("mock %s callback unauthorized", callbackType))
		return MockApplicationCallbackError
	}

	ctx.GasMeter().ConsumeGas(500000, fmt.Sprintf("mock %s callback success", callbackType))
	return nil
}
