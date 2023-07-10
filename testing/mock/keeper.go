package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	callbacktypes "github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
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
	AckCallbackCounter        *types.CallbackCounter
	TimeoutCallbackCounter    *types.CallbackCounter
	RecvPacketCallbackCounter *types.CallbackCounter
}

// NewKeeper creates a new mock Keeper.
func NewMockKeeper() MockKeeper {
	return MockKeeper{
		MockContractKeeper: MockContractKeeper{
			AckCallbackCounter:        types.NewCallbackCounter(),
			TimeoutCallbackCounter:    types.NewCallbackCounter(),
			RecvPacketCallbackCounter: types.NewCallbackCounter(),
		},
	}
}

// IBCAcknowledgementPacketCallback returns nil if the gas meter has greater than
// or equal to 100000 gas remaining. Otherwise, it returns an out of gas error.
// This function also consumes 100000 gas, or the remaining gas if less than 100000.
func (k MockContractKeeper) IBCAcknowledgementPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	contractAddr string,
) error {
	if ctx.GasMeter().GasRemaining() < 100000 {
		k.AckCallbackCounter.IncrementFailure()
		ctx.GasMeter().ConsumeGas(ctx.GasMeter().GasRemaining(), "mock ack callback failure")
		return ibcerrors.ErrOutOfGas
	}

	k.AckCallbackCounter.IncrementSuccess()
	ctx.GasMeter().ConsumeGas(100000, "mock ack callback success")
	return nil
}

// IBCPacketTimeoutCallback returns nil if the gas meter has greater than
// or equal to 100000 gas remaining. Otherwise, it returns an out of gas error.
// This function also consumes 100000 gas, or the remaining gas if less than 100000.
func (k MockContractKeeper) IBCPacketTimeoutCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	contractAddr string,
) error {
	if ctx.GasMeter().GasRemaining() < 100000 {
		k.TimeoutCallbackCounter.IncrementFailure()
		ctx.GasMeter().ConsumeGas(ctx.GasMeter().GasRemaining(), "mock timeout callback failure")
		return ibcerrors.ErrOutOfGas
	}

	k.TimeoutCallbackCounter.IncrementSuccess()
	ctx.GasMeter().ConsumeGas(100000, "mock timeout callback success")
	return nil
}

// IBCReceivePacketCallback returns nil if the gas meter has greater than
// or equal to 100000 gas remaining. Otherwise, it returns an out of gas error.
// This function also consumes 100000 gas, or the remaining gas if less than 100000.
func (k MockContractKeeper) IBCReceivePacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	ackResult channeltypes.Acknowledgement,
	relayer sdk.AccAddress,
	contractAddr string,
) error {
	if ctx.GasMeter().GasRemaining() < 100000 {
		k.RecvPacketCallbackCounter.IncrementFailure()
		ctx.GasMeter().ConsumeGas(ctx.GasMeter().GasRemaining(), "mock recv packet callback failure")
		return ibcerrors.ErrOutOfGas
	}

	k.RecvPacketCallbackCounter.IncrementSuccess()
	ctx.GasMeter().ConsumeGas(100000, "mock recv packet callback success")
	return nil
}
