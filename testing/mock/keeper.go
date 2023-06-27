package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	callbacktypes "github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// Keeper implements callbacktypes.ContractKeeper
var _ callbacktypes.ContractKeeper = MockKeeper{}

// This is a mock keeper used for testing. It is not wired up to any modules.
// It implements the interface functions expected by the ibccallbacks middleware
// so that it can be tested with simapp.
type MockKeeper struct{}

// NewKeeper creates a new mock Keeper.
func NewMockKeeper() MockKeeper {
	return MockKeeper{}
}

// IBCAcknowledgementPacketCallback returns nil.
func (MockKeeper) IBCAcknowledgementPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	customMsg []byte,
	ackResult channeltypes.Acknowledgement,
	relayer sdk.AccAddress,
	contractAddr string,
) error {
	return nil
}

// IBCPacketTimeoutCallback returns nil.
func (MockKeeper) IBCPacketTimeoutCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	contractAddr string,
) error {
	return nil
}

// IBCReceivePacketCallback returns nil.
func (MockKeeper) IBCReceivePacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	customMsg []byte,
	ackResult channeltypes.Acknowledgement,
	relayer sdk.AccAddress,
	contractAddr string,
) error {
	return nil
}
