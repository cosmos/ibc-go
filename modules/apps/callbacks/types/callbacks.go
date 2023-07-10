package types

import (
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// PacketUnmarshalerIBCModule is an interface that combines the IBCModule and PacketDataUnmarshaler
// interfaces to assert that the underlying application supports both.
type PacketUnmarshalerIBCModule interface {
	porttypes.IBCModule
	porttypes.PacketDataUnmarshaler
}

// CallbackData is the callback data parsed from the packet.
type CallbackData struct {
	ContractAddr string
	GasLimit     uint64
}

// GetSourceCallbackData parses the packet data and returns the source callback data. It ensures
// that the remaining gas is greater than the gas limit specified in the packet data.
func GetSourceCallbackData(app PacketUnmarshalerIBCModule, packet channeltypes.Packet, remainingGas uint64) (CallbackData, error) {
	// unmarshal packet data
	unmarshaledData, err := app.UnmarshalPacketData(packet.Data)
	if err != nil {
		return CallbackData{}, err
	}

	callbackData, ok := unmarshaledData.(ibcexported.CallbackPacketData)
	if !ok {
		return CallbackData{}, ErrNotCallbackPacketData
	}

	gasLimit := callbackData.GetSourceUserDefinedGasLimit()
	if gasLimit == 0 || gasLimit > remainingGas {
		gasLimit = remainingGas
	}

	return CallbackData{
		ContractAddr: callbackData.GetSourceCallbackAddress(),
		GasLimit:     gasLimit,
	}, nil
}

// GetDestCallbackData parses the packet data and returns the source callback data. It ensures
// that the remaining gas is greater than the gas limit specified in the packet data.
func GetDestCallbackData(app PacketUnmarshalerIBCModule, packet channeltypes.Packet, remainingGas uint64) (CallbackData, error) {
	// unmarshal packet data
	unmarshaledData, err := app.UnmarshalPacketData(packet.Data)
	if err != nil {
		return CallbackData{}, err
	}

	callbackData, ok := unmarshaledData.(ibcexported.CallbackPacketData)
	if !ok {
		return CallbackData{}, ErrNotCallbackPacketData
	}

	gasLimit := callbackData.GetDestUserDefinedGasLimit()
	if gasLimit == 0 || gasLimit > remainingGas {
		gasLimit = remainingGas
	}

	return CallbackData{
		ContractAddr: callbackData.GetDestCallbackAddress(),
		GasLimit:     gasLimit,
	}, nil
}
