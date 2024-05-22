package types

import (
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

/*

ADR-8 implementation

The Memo is used to ensure that the callback is desired by the user. This allows a user to send a packet to an ADR-8 enabled contract.

The Memo format is defined like so:

```json
{
	// ... other memo fields we don't care about
	"src_callback": {
		"address": {stringCallbackAddress},

		// optional fields
		"gas_limit": {stringForCallback}
	},
	"dest_callback": {
		"address": {stringCallbackAddress},

		// optional fields
		"gas_limit": {stringForCallback}
	}
}
```

We will pass the packet sender info (if available) to the contract keeper for source callback executions. This will allow the contract
keeper to verify that the packet sender is the same as the callback address if desired.

*/

// CallbacksCompatibleModule is an interface that combines the IBCModule and PacketDataUnmarshaler
// interfaces to assert that the underlying application supports both.
type CallbacksCompatibleModule interface {
	porttypes.IBCModule
	porttypes.PacketDataUnmarshaler
}

// CallbackData is the callback data parsed from the packet.
type CallbackData struct {
	// CallbackAddress is the address of the callback actor.
	CallbackAddress string
	// ExecutionGasLimit is the gas limit which will be used for the callback execution.
	ExecutionGasLimit uint64
	// SenderAddress is the sender of the packet. This is passed to the contract keeper
	// to verify that the packet sender is the same as the callback address if desired.
	// This address is empty during destination callback execution.
	// This address may be empty if the sender is unknown or undefined.
	SenderAddress string
	// CommitGasLimit is the gas needed to commit the callback even if the callback
	// execution fails due to out of gas.
	// This parameter is only used in event emissions, or logging.
	CommitGasLimit uint64
}

// GetSourceCallbackData parses the packet data and returns the source callback data.
func GetSourceCallbackData(
	ctx sdk.Context,
	packetDataUnmarshaler porttypes.PacketDataUnmarshaler,
	packet channeltypes.Packet,
	maxGas uint64,
) (CallbackData, error) {
	packetData, err := packetDataUnmarshaler.UnmarshalPacketData(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetData())
	if err != nil {
		return CallbackData{}, errorsmod.Wrap(ErrCannotUnmarshalPacketData, err.Error())
	}

	return getCallbackData(packetData, packet.GetSourcePort(), ctx.GasMeter().GasRemaining(), maxGas, SourceCallbackKey)
}

// GetDestCallbackData parses the packet data and returns the destination callback data.
func GetDestCallbackData(
	ctx sdk.Context,
	packetDataUnmarshaler porttypes.PacketDataUnmarshaler,
	packet channeltypes.Packet, maxGas uint64,
) (CallbackData, error) {
	packetData, err := packetDataUnmarshaler.UnmarshalPacketData(ctx, packet.GetDestPort(), packet.GetDestChannel(), packet.GetData())
	if err != nil {
		return CallbackData{}, errorsmod.Wrap(ErrCannotUnmarshalPacketData, err.Error())
	}

	return getCallbackData(packetData, packet.GetSourcePort(), ctx.GasMeter().GasRemaining(), maxGas, DestinationCallbackKey)
}

// getCallbackData parses the packet data and returns the callback data.
// It also checks that the remaining gas is greater than the gas limit specified in the packet data.
// The addressGetter and gasLimitGetter functions are used to retrieve the callback
// address and gas limit from the callback data.
func getCallbackData(
	packetData interface{},
	srcPortID string,
	remainingGas,
	maxGas uint64, callbackKey string,
) (CallbackData, error) {
	packetDataProvider, ok := packetData.(ibcexported.PacketDataProvider)
	if !ok {
		return CallbackData{}, ErrNotPacketDataProvider
	}

	callbackData, ok := packetDataProvider.GetCustomPacketData(callbackKey).(map[string]interface{})
	if callbackData == nil || !ok {
		return CallbackData{}, ErrCallbackKeyNotFound
	}

	// get the callback address from the callback data
	callbackAddress := getCallbackAddress(callbackData)
	if strings.TrimSpace(callbackAddress) == "" {
		return CallbackData{}, ErrCallbackAddressNotFound
	}

	// retrieve packet sender from packet data if possible and if needed
	var packetSender string
	if callbackKey == SourceCallbackKey {
		packetData, ok := packetData.(ibcexported.PacketData)
		if ok {
			packetSender = packetData.GetPacketSender(srcPortID)
		}
	}

	// get the gas limit from the callback data
	executionGasLimit, commitGasLimit := computeExecAndCommitGasLimit(callbackData, remainingGas, maxGas)

	return CallbackData{
		CallbackAddress:   callbackAddress,
		ExecutionGasLimit: executionGasLimit,
		SenderAddress:     packetSender,
		CommitGasLimit:    commitGasLimit,
	}, nil
}

func computeExecAndCommitGasLimit(callbackData map[string]interface{}, remainingGas, maxGas uint64) (uint64, uint64) {
	// get the gas limit from the callback data
	commitGasLimit := getUserDefinedGasLimit(callbackData)

	// ensure user defined gas limit does not exceed the max gas limit
	if commitGasLimit == 0 || commitGasLimit > maxGas {
		commitGasLimit = maxGas
	}

	// account for the remaining gas in the context being less than the desired gas limit for the callback execution
	// in this case, the callback execution may be retried upon failure
	executionGasLimit := commitGasLimit
	if remainingGas < executionGasLimit {
		executionGasLimit = remainingGas
	}

	return executionGasLimit, commitGasLimit
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

// AllowRetry returns true if the callback execution gas limit is less than the commit gas limit.
func (c CallbackData) AllowRetry() bool {
	return c.ExecutionGasLimit < c.CommitGasLimit
}
