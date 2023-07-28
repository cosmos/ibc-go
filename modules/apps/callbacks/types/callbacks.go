package types

import (
	"strconv"

	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

/*

ADR-8 implementation

The Memo is used to ensure that the callback is desired by the user. This allows a user to send a packet to an ADR-8 enabled contract.

The Memo format is defined like so:

```json
{
	// ... other memo fields we don't care about
	"src_callback": {
		"address": {stringContractAddress},

		// optional fields
		"gas_limit": {stringForCallback}
	},
	"dest_callback": {
		"address": {stringContractAddress},

		// optional fields
		"gas_limit": {stringForCallback}
	}
}
```

We will pass the packet sender info (if available) to the contract keeper for source callback executions. This will allow the contract
keeper to verify that the packet sender is the same as the contract address if desired.

*/

// CallbacksCompatibleModule is an interface that combines the IBCModule and PacketDataUnmarshaler
// interfaces to assert that the underlying application supports both.
type CallbacksCompatibleModule interface {
	porttypes.IBCModule
	porttypes.PacketDataUnmarshaler
}

// CallbackData is the callback data parsed from the packet.
type CallbackData struct {
	// ContractAddr is the address of the callback contract.
	ContractAddr string
	// GasLimit is the gas limit which will be used for the callback execution.
	GasLimit uint64
	// SenderAddr is the sender of the packet. This is passed to the contract keeper
	// to verify that the packet sender is the same as the contract address if desired.
	// This address is empty during destination callback execution.
	// This address may be empty if the sender is unknown or undefined.
	SenderAddr string
	// CommitGasLimit is the gas needed to commit the callback even if the callback
	// execution fails due to out of gas.
	// This parameter is only used in event emissions, or logging.
	CommitGasLimit uint64
}

// GetSourceCallbackData parses the packet data and returns the source callback data.
// It also checks that the remaining gas is greater than the gas limit specified in the packet data.
func GetSourceCallbackData(
	packetDataUnmarshaler porttypes.PacketDataUnmarshaler,
	packet ibcexported.PacketI, remainingGas uint64, maxGas uint64,
) (CallbackData, bool, error) {
	return getCallbackData(packetDataUnmarshaler, packet, remainingGas, maxGas, SourceCallbackMemoKey)
}

// GetDestCallbackData parses the packet data and returns the destination callback data.
// It also checks that the remaining gas is greater than the gas limit specified in the packet data.
func GetDestCallbackData(
	packetDataUnmarshaler porttypes.PacketDataUnmarshaler,
	packet ibcexported.PacketI, remainingGas uint64, maxGas uint64,
) (CallbackData, bool, error) {
	return getCallbackData(packetDataUnmarshaler, packet, remainingGas, maxGas, DestCallbackMemoKey)
}

// getCallbackData parses the packet data and returns the callback data.
// It also checks that the remaining gas is greater than the gas limit specified in the packet data.
// The addressGetter and gasLimitGetter functions are used to retrieve the callback
// address and gas limit from the callback data.
func getCallbackData(
	packetDataUnmarshaler porttypes.PacketDataUnmarshaler,
	packet ibcexported.PacketI, remainingGas uint64,
	maxGas uint64, callbackKey string,
) (CallbackData, bool, error) {
	// unmarshal packet data
	unmarshaledData, err := packetDataUnmarshaler.UnmarshalPacketData(packet.GetData())
	if err != nil {
		return CallbackData{}, false, err
	}

	packetDataProvider, ok := unmarshaledData.(ibcexported.PacketDataProvider)
	if !ok {
		return CallbackData{}, false, ErrNotAdditionalPacketDataProvider
	}

	callbackData, ok := packetDataProvider.GetCustomPacketData(callbackKey).(map[string]interface{})
	if callbackData == nil || !ok {
		return CallbackData{}, false, ErrCallbackMemoKeyNotFound
	}

	// if the relayer did not specify enough gas to meet the minimum of the
	// user defined gas limit and the max allowed gas limit, the callback execution
	// may be retried
	var allowRetry bool

	// get the gas limit from the callback data
	gasLimit := getUserDefinedGasLimit(callbackData)

	// ensure user defined gas limit does not exceed the max gas limit
	if gasLimit == 0 || gasLimit > maxGas {
		gasLimit = maxGas
	}

	// account for the remaining gas in the context being less than the desired gas limit for the callback execution
	// in this case, the callback execution may be retried upon failure
	commitGasLimit := gasLimit
	if remainingGas < gasLimit {
		gasLimit = remainingGas
		allowRetry = true
	}

	// retrieve packet sender from packet data if possible and if needed
	var packetSender string
	if callbackKey == SourceCallbackMemoKey {
		packetSenderRetriever, ok := unmarshaledData.(ibcexported.PacketSenderRetriever)
		if ok {
			packetSender = packetSenderRetriever.GetPacketSender(packet.GetSourcePort())
		}
	}

	return CallbackData{
		ContractAddr:   getCallbackAddress(callbackData),
		GasLimit:       gasLimit,
		SenderAddr:     packetSender,
		CommitGasLimit: commitGasLimit,
	}, allowRetry, nil
}

// getUserDefinedGasLimit returns the custom gas limit provided for callbacks if it is
// in the callback data. It is assumed that callback data is not nil.
// If no gas limit is specified or the gas limit is improperly formatted, 0 is returned.
//
// The memo is expected to specify the user defined gas limit in the following format:
// { "{callbackKey}": { ... , "gas_limit": {stringForCallback} }
//
// Note: the user defined gas limit must be set as a string and not a json number.
func getUserDefinedGasLimit(callbackData map[string]interface{}) uint64 {
	// the gas limit must be specified as a string and not a json number
	gasLimit, ok := callbackData[UserDefinedGasLimitKey].(string)
	if !ok {
		return 0
	}

	userGas, err := strconv.ParseUint(gasLimit, 10, 64)
	if err != nil {
		return 0
	}

	return userGas
}

// getCallbackAddress returns the callback address if it is specified in the callback data.
// It is assumed that callback data is not nil.
// If no callback address is specified or the memo is improperly formatted, an empty string is returned.
//
// The memo is expected to contain the callback address in the following format:
// { "{callbackKey}": { "address": {stringCallbackAddress}}
//
// ADR-8 middleware should callback on the returned address if it is a PacketActor
// (i.e. smart contract that accepts IBC callbacks).
func getCallbackAddress(callbackData map[string]interface{}) string {
	callbackAddress, ok := callbackData[CallbackAddressKey].(string)
	if !ok {
		return ""
	}

	return callbackAddress
}
