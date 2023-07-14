package types

import (
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// PacketUnmarshalerIBCModule is an interface that combines the IBCModule and PacketInfoProvider
// interfaces to assert that the underlying application supports both.
type PacketInfoProviderIBCModule interface {
	porttypes.IBCModule
	porttypes.PacketInfoProvider
}

// CallbackData is the callback data parsed from the packet.
type CallbackData struct {
	ContractAddr string
	GasLimit     uint64
}

// GetSourceCallbackData parses the packet data and returns the source callback data. It ensures
// that the remaining gas is greater than the gas limit specified in the packet data.
func GetSourceCallbackData(
	packetInfoProvider porttypes.PacketInfoProvider,
	packetData []byte, remainingGas uint64,
) (CallbackData, error) {
	addressGetter := func(callbackData ibcexported.CallbackPacketData) string {
		return callbackData.GetSourceCallbackAddress()
	}
	gasLimitGetter := func(callbackData ibcexported.CallbackPacketData) uint64 {
		return callbackData.GetSourceUserDefinedGasLimit()
	}
	return getCallbackData(packetInfoProvider, packetData, remainingGas, addressGetter, gasLimitGetter)
}

// GetDestCallbackData parses the packet data and returns the source callback data. It ensures
// that the remaining gas is greater than the gas limit specified in the packet data.
func GetDestCallbackData(
	packetInfoProvider porttypes.PacketInfoProvider,
	packetData []byte, remainingGas uint64,
) (CallbackData, error) {
	addressGetter := func(callbackData ibcexported.CallbackPacketData) string {
		return callbackData.GetDestCallbackAddress()
	}
	gasLimitGetter := func(callbackData ibcexported.CallbackPacketData) uint64 {
		return callbackData.GetDestUserDefinedGasLimit()
	}
	return getCallbackData(packetInfoProvider, packetData, remainingGas, addressGetter, gasLimitGetter)
}

// getCallbackData parses the packet data and returns the callback data. It ensures
// that the remaining gas is greater than the gas limit specified in the packet data.
// The addressGetter and gasLimitGetter functions are used to retrieve the callback
// address and gas limit from the callback data.
func getCallbackData(
	packetInfoProvider porttypes.PacketInfoProvider,
	packetData []byte, remainingGas uint64,
	addressGetter func(ibcexported.CallbackPacketData) string,
	gasLimitGetter func(ibcexported.CallbackPacketData) uint64,
) (CallbackData, error) {
	// unmarshal packet data
	unmarshaledData, err := packetInfoProvider.UnmarshalPacketData(packetData)
	if err != nil {
		return CallbackData{}, err
	}

	callbackData, ok := unmarshaledData.(ibcexported.CallbackPacketData)
	if !ok {
		return CallbackData{}, ErrNotCallbackPacketData
	}

	gasLimit := gasLimitGetter(callbackData)
	if gasLimit == 0 || gasLimit > remainingGas {
		gasLimit = remainingGas
	}

	return CallbackData{
		ContractAddr: addressGetter(callbackData),
		GasLimit:     gasLimit,
	}, nil
}
