package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
)

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

// GetCallbackData is a wrapper around getCallbackData to allow the function to be directly called in tests.
func GetCallbackData(
	ctx sdk.Context,
	packetDataUnmarshaler porttypes.PacketDataUnmarshaler,
	packet channeltypes.Packet,
	remainingGas,
	maxGas uint64, callbackKey string,
) (CallbackData, error) {
	// TODO(jim): Probably just refactor single occurrence of this in tests
	if callbackKey == DestinationCallbackKey {
		return GetDestCallbackData(ctx, packetDataUnmarshaler, packet, maxGas)
	}
	return GetSourceCallbackData(ctx, packetDataUnmarshaler, packet, maxGas)
}

// GetCallbackAddress is a wrapper around getCallbackAddress to allow the function to be directly called in tests.
func GetCallbackAddress(callbackData map[string]interface{}) string {
	return getCallbackAddress(callbackData)
}

// GetUserDefinedGasLimit is a wrapper around getUserDefinedGasLimit to allow the function to be directly called in tests.
func GetUserDefinedGasLimit(callbackData map[string]interface{}) uint64 {
	return getUserDefinedGasLimit(callbackData)
}
