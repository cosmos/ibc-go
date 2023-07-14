package mock

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	callbacktypes "github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/ibc-go/v7/testing/mock/types"
)

// MockKeeper implements callbacktypes.ContractKeeper
var _ callbacktypes.ContractKeeper = (*MockKeeper)(nil)

// MockKeeper can be used to mock the expected keepers needed for testing.
//
// MockKeeper currently mocks the following interfaces:
//   - callbacktypes.ContractKeeper
type MockKeeper struct {
	MockContractKeeper
}

// This is a mock keeper used for testing. It is not wired up to any modules.
// It implements the interface functions expected by the ibccallbacks middleware
// so that it can be tested with simapp.
type MockContractKeeper struct {
	SendPacketCallbackCounter           *types.CallbackCounter
	AckCallbackCounter                  *types.CallbackCounter
	TimeoutCallbackCounter              *types.CallbackCounter
	WriteAcknowledgementCallbackCounter *types.CallbackCounter
}

// NewKeeper creates a new mock Keeper.
func NewMockKeeper() MockKeeper {
	return MockKeeper{
		MockContractKeeper: MockContractKeeper{
			SendPacketCallbackCounter:           types.NewCallbackCounter(),
			AckCallbackCounter:                  types.NewCallbackCounter(),
			TimeoutCallbackCounter:              types.NewCallbackCounter(),
			WriteAcknowledgementCallbackCounter: types.NewCallbackCounter(),
		},
	}
}

// IBCPacketSendCallback returns nil if the gas meter has greater than
// or equal to 100000 gas remaining. Otherwise, it returns an out of gas error.
// This function also consumes 100000 gas, or the remaining gas if less than 100000.
// This function panics if the gas remaining is less than 10000.
func (k MockContractKeeper) IBCSendPacketCallback(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	packetData []byte,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallbacks(ctx, callbacktypes.CallbackTypeSendPacket, k.SendPacketCallbackCounter)
}

// IBCOnAcknowledgementPacketCallback returns nil if the gas meter has greater than
// or equal to 100000 gas remaining. Otherwise, it returns an out of gas error.
// This function also consumes 100000 gas, or the remaining gas if less than 100000.
// This function panics if the gas remaining is less than 10000.
func (k MockContractKeeper) IBCOnAcknowledgementPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallbacks(ctx, callbacktypes.CallbackTypeAcknowledgement, k.AckCallbackCounter)
}

// IBCOnTimeoutPacketCallback returns nil if the gas meter has greater than
// or equal to 100000 gas remaining. Otherwise, it returns an out of gas error.
// This function also consumes 100000 gas, or the remaining gas if less than 100000.
// This function panics if the gas remaining is less than 10000.
func (k MockContractKeeper) IBCOnTimeoutPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallbacks(ctx, callbacktypes.CallbackTypeTimeoutPacket, k.TimeoutCallbackCounter)
}

// IBCWriteAcknowledgementCallback returns nil if the gas meter has greater than
// or equal to 100000 gas remaining. Otherwise, it returns an out of gas error.
// This function also consumes 100000 gas, or the remaining gas if less than 100000.
// This function panics if the gas remaining is less than 10000.
func (k MockContractKeeper) IBCWriteAcknowledgementCallback(
	ctx sdk.Context,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
	contractAddress,
	packetSenderAddress string,
) error {
	return k.processMockCallbacks(ctx, callbacktypes.CallbackTypeWriteAcknowledgement, k.WriteAcknowledgementCallbackCounter)
}

func (k MockContractKeeper) processMockCallbacks(
	ctx sdk.Context,
	callbackType callbacktypes.CallbackType,
	callbackCounter *types.CallbackCounter,
) error {
	gasRemaining := ctx.GasMeter().GasRemaining()
	if gasRemaining < 10000 {
		// panic if gas remaining is less than 10000, for tests
		ctx.GasMeter().ConsumeGas(ctx.GasMeter().GasRemaining(), fmt.Sprintf("mock %s callback panic", callbackType))
		callbackCounter.IncrementFailure()
		panic("mock recv packet callback failure")
	} else if gasRemaining < 100000 {
		// error if gas remaining is less than 100000, for tests
		callbackCounter.IncrementFailure()
		ctx.GasMeter().ConsumeGas(ctx.GasMeter().GasRemaining(), fmt.Sprintf("mock %s callback failure", callbackType))
		return ibcerrors.ErrOutOfGas
	}

	callbackCounter.IncrementSuccess()
	ctx.GasMeter().ConsumeGas(100000, fmt.Sprintf("mock %s callback success", callbackType))
	return nil
}
