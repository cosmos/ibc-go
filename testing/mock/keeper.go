package mock

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	callbacktypes "github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/ibc-go/v7/testing/mock/types"
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
// so that it can be tested with simapp.
type ContractKeeper struct {
	SendPacketCallbackCounter           *types.CallbackCounter
	AckCallbackCounter                  *types.CallbackCounter
	TimeoutCallbackCounter              *types.CallbackCounter
	WriteAcknowledgementCallbackCounter *types.CallbackCounter
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
			SendPacketCallbackCounter:           types.NewCallbackCounter(),
			AckCallbackCounter:                  types.NewCallbackCounter(),
			TimeoutCallbackCounter:              types.NewCallbackCounter(),
			WriteAcknowledgementCallbackCounter: types.NewCallbackCounter(),
		},
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
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeSendPacket, k.SendPacketCallbackCounter, packetSenderAddress)
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
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeAcknowledgement, k.AckCallbackCounter, packetSenderAddress)
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
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeTimeoutPacket, k.TimeoutCallbackCounter, packetSenderAddress)
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
	return k.processMockCallback(ctx, callbacktypes.CallbackTypeWriteAcknowledgement, k.WriteAcknowledgementCallbackCounter, "")
}

// processMockCallback returns nil if the gas meter has greater than or equal to 500000 gas remaining.
// This function consumes 500000 gas, or the remaining gas if less than 500000.
// This function oog panics if the gas remaining is less than 400000.
func (k Keeper) processMockCallback(
	ctx sdk.Context,
	callbackType callbacktypes.CallbackType,
	callbackCounter *types.CallbackCounter,
	authAddress string,
) error {
	gasRemaining := ctx.GasMeter().GasRemaining()
	k.IncrementStatefulCounter(ctx)

	if gasRemaining < 400000 {
		callbackCounter.IncrementFailure()
		// consume gas will panic since we attempt to consume 500_000 gas, for tests
		ctx.GasMeter().ConsumeGas(500000, fmt.Sprintf("mock %s callback panic", callbackType))
	} else if gasRemaining < 500000 {
		callbackCounter.IncrementFailure()
		ctx.GasMeter().ConsumeGas(gasRemaining, fmt.Sprintf("mock %s callback failure", callbackType))
		return MockApplicationCallbackError
	}

	if authAddress == MockCallbackUnauthorizedAddress {
		callbackCounter.IncrementFailure()
		ctx.GasMeter().ConsumeGas(500000, fmt.Sprintf("mock %s callback unauthorized", callbackType))
		return MockApplicationCallbackError
	}

	callbackCounter.IncrementSuccess()
	ctx.GasMeter().ConsumeGas(500000, fmt.Sprintf("mock %s callback success", callbackType))
	return nil
}
