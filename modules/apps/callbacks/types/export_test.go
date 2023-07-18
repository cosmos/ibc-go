package types

import (
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

// GetCallbackData is a wrapper around getCallbackData to allow the function to be directly called in tests.
func GetCallbackData(
	packetInfoProvider porttypes.PacketInfoProvider,
	packetData []byte, remainingGas uint64, maxGas uint64,
	addressGetter func(ibcexported.CallbackPacketData) string,
	gasLimitGetter func(ibcexported.CallbackPacketData) uint64,
) (CallbackData, bool, error) {
	return getCallbackData(packetInfoProvider, packetData, remainingGas, maxGas, addressGetter, gasLimitGetter)
}
