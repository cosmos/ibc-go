package types

import (
	"encoding/json"
	"strconv"
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
	"callback": {
		"src_callback_address": {contractAddrOnSourceChain},

		// optional fields
		"gas_limit": {stringForCallback}
	}
}
```

*/

// GetSourceCallbackAddress returns the source callback address provided in the packet data memo.
// If no callback address is specified, an empty string is returned.
//
// The memo is expected to specify the callback address in the following format:
// { "callback": { "src_callback_address": {contractAddrOnSourceChain}}
//
// ADR-8 middleware should callback on the returned address if it is a PacketActor
// (i.e. smart contract that accepts IBC callbacks).
func (iapd InterchainAccountPacketData) GetSourceCallbackAddress() string {
	callbackData := iapd.getCallbackData()
	if callbackData == nil {
		return ""
	}

	callbackAddr, ok := callbackData["src_callback_address"].(string)
	if !ok {
		return ""
	}

	return callbackAddr
}

// GetDestCallbackAddress returns an empty string. Destination callback addresses
// are not supported for ICS 27. This feature is natively supported by
// interchain accounts host submodule transaction execution.
func (iapd InterchainAccountPacketData) GetDestCallbackAddress() string {
	return ""
}

// UserDefinedGasLimit returns the custom gas limit provided in the packet data memo.
//
// The memo is expected to specify the callback address in the following format:
// { "callback": { ... , "gas_limit": {stringForCallback} }
//
// If no gas limit is specified, 0 is returned.
func (iapd InterchainAccountPacketData) UserDefinedGasLimit() uint64 {
	callbackData := iapd.getCallbackData()
	if callbackData == nil {
		return 0
	}

	// json number won't be unmarshaled as a uint64, so we a use string instead
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

// getCallbackData returns the memo as `map[string]interface{}` so that it can be
// interpreted as a json object with keys.
func (iapd InterchainAccountPacketData) getCallbackData() map[string]interface{} {
	if len(iapd.Memo) == 0 {
		return nil
	}

	jsonObject := make(map[string]interface{})
	err := json.Unmarshal([]byte(iapd.Memo), &jsonObject)
	if err != nil {
		return nil
	}

	callbackData, ok := jsonObject["callback"].(map[string]interface{})
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
