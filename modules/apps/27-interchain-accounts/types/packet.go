package types

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// MaxMemoCharLength defines the maximum length for the InterchainAccountPacketData memo field
const MaxMemoCharLength = 256

var (
	// DefaultRelativePacketTimeoutHeight is the default packet timeout height (in blocks) relative
	// to the current block height of the counterparty chain provided by the client state. The
	// timeout is disabled when set to 0.
	DefaultRelativePacketTimeoutHeight = "0-1000"

	// DefaultRelativePacketTimeoutTimestamp is the default packet timeout timestamp (in nanoseconds)
	// relative to the current block timestamp of the counterparty chain provided by the client
	// state. The timeout is disabled when set to 0. The default is currently set to a 10 minute
	// timeout.
	DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())
)

var _ exported.CallbackPacketData = (*InterchainAccountPacketData)(nil)

// ValidateBasic performs basic validation of the interchain account packet data.
// The memo may be empty.
func (iapd InterchainAccountPacketData) ValidateBasic() error {
	if iapd.Type == UNSPECIFIED {
		return errorsmod.Wrap(ErrInvalidOutgoingData, "packet data type cannot be unspecified")
	}

	if len(iapd.Data) == 0 {
		return errorsmod.Wrap(ErrInvalidOutgoingData, "packet data cannot be empty")
	}

	if len(iapd.Memo) > MaxMemoCharLength {
		return errorsmod.Wrapf(ErrInvalidOutgoingData, "packet data memo cannot be greater than %d characters", MaxMemoCharLength)
	}

	return nil
}

// GetBytes returns the JSON marshalled interchain account packet data.
func (iapd InterchainAccountPacketData) GetBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&iapd))
}

/*

ADR-8 CallbackPacketData implementation

InterchainAccountPacketData implements CallbackPacketData interface. This will allow middlewares targeting specific VMs
to retrieve the desired callback address for the ICA packet on the source chain. Destination callback addresses are not
supported for ICS 27.

The Memo is used to set the desired callback addresses.

The Memo format is defined like so:

```json
{
	// ... other memo fields we don't care about
	"src_callback": {
		"address": {stringForContractAddress},

		// optional fields
		"gas_limit": {stringForGasLimit},
	}
}
```

*/

// GetSourceCallbackAddress returns the source callback address if it is specified in the packet data memo.
// If no callback address is specified or the memo is improperly formatted, an empty string is returned.
//
// The memo is expected to contain the source callback address in the following format:
// { "src_callback": { "address": {stringCallbackAddress}}
//
// ADR-8 middleware should callback on the returned address if it is a PacketActor
// (i.e. smart contract that accepts IBC callbacks).
func (iapd InterchainAccountPacketData) GetSourceCallbackAddress() string {
	callbackData := iapd.getCallbackData("src_callback")
	if callbackData == nil {
		return ""
	}

	callbackAddress, ok := callbackData["address"].(string)
	if !ok {
		return ""
	}

	return callbackAddress
}

// GetDestCallbackAddress returns an empty string. Destination callback addresses
// are not supported for ICS 27. This feature is natively supported by
// interchain accounts host submodule transaction execution.
func (iapd InterchainAccountPacketData) GetDestCallbackAddress() string {
	return ""
}

// GetSourceUserDefinedGasLimit returns the custom gas limit provided for source callbacks
// if it is specified in the packet data memo.
// If no gas limit is specified or the gas limit is improperly formatted, 0 is returned.
//
// The memo is expected to specify the user defined gas limit in the following format:
// { "src_callback": { ... , "gas_limit": {stringForGasLimit} }
//
// Note: the user defined gas limit must be set as a string and not a json number.
func (iapd InterchainAccountPacketData) GetSourceUserDefinedGasLimit() uint64 {
	callbackData := iapd.getCallbackData("src_callback")
	if callbackData == nil {
		return 0
	}

	// the gas limit must be specified as a string and not a json number
	gasLimit, ok := callbackData["gas_limit"].(string)
	if !ok {
		return 0
	}

	userGas, err := strconv.ParseUint(gasLimit, 10, 64)
	if err != nil {
		return 0
	}

	return userGas
}

// GetDestUserDefinedGasLimit returns 0. Destination callbacks are not supported for ICS 27.
// This feature is natively supported by interchain accounts host submodule transaction execution.
func (iapd InterchainAccountPacketData) GetDestUserDefinedGasLimit() uint64 {
	return 0
}

// GetPacketSender returns the sender address of the packet.
func (iapd InterchainAccountPacketData) GetPacketSender(srcPortID, srcChannelID string) string {
	icaOwner, found := strings.CutPrefix(srcPortID, ControllerPortPrefix)
	if !found {
		return ""
	}
	return icaOwner
}

// GetPacketReceiver returns the empty string because destination callbacks are not supported for ICS 27.
func (iapd InterchainAccountPacketData) GetPacketReceiver(dstPortID, dstChannelID string) string {
	return ""
}

// getCallbackData returns the memo as `map[string]interface{}` so that it can be
// interpreted as a json object with keys.
func (iapd InterchainAccountPacketData) getCallbackData(callbackKey string) map[string]interface{} {
	if len(iapd.Memo) == 0 {
		return nil
	}

	jsonObject := make(map[string]interface{})
	err := json.Unmarshal([]byte(iapd.Memo), &jsonObject)
	if err != nil {
		return nil
	}

	callbackData, ok := jsonObject[callbackKey].(map[string]interface{})
	if !ok {
		return nil
	}

	return callbackData
}

// GetBytes returns the JSON marshalled interchain account CosmosTx.
func (ct CosmosTx) GetBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&ct))
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (ct CosmosTx) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	for _, protoAny := range ct.Messages {
		err := unpacker.UnpackAny(protoAny, new(sdk.Msg))
		if err != nil {
			return err
		}
	}

	return nil
}
