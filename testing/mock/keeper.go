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
	AckCallbackCounter        types.CallbackCounter
	TimeoutCallbackCounter    types.CallbackCounter
	RecvPacketCallbackCounter types.CallbackCounter
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
func (k MockContractKeeper) IBCAcknowledgementPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	customMsg []byte,
	ackResult channeltypes.Acknowledgement,
	relayer sdk.AccAddress,
	contractAddr string,
) error {
	if ctx.GasMeter().GasRemaining() < 100000 {
		k.AckCallbackCounter.IncrementFailure()
		return ibcerrors.ErrOutOfGas
	}

	k.AckCallbackCounter.IncrementSuccess()
	return nil
}

// IBCPacketTimeoutCallback returns nil if the gas meter has greater than
// or equal to 100000 gas remaining. Otherwise, it returns an out of gas error.
func (k MockContractKeeper) IBCPacketTimeoutCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	contractAddr string,
) error {
	if ctx.GasMeter().GasRemaining() < 100000 {
		k.TimeoutCallbackCounter.IncrementFailure()
		return ibcerrors.ErrOutOfGas
	}

	k.TimeoutCallbackCounter.IncrementSuccess()
	return nil
}

// IBCReceivePacketCallback returns nil if the gas meter has greater than
// or equal to 100000 gas remaining. Otherwise, it returns an out of gas error.
func (k MockContractKeeper) IBCReceivePacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	customMsg []byte,
	ackResult channeltypes.Acknowledgement,
	relayer sdk.AccAddress,
	contractAddr string,
) error {
	if ctx.GasMeter().GasRemaining() < 100000 {
		k.RecvPacketCallbackCounter.IncrementFailure()
		return ibcerrors.ErrOutOfGas
	}

	k.RecvPacketCallbackCounter.IncrementSuccess()
	return nil
}
