package types

import (
	"encoding/base64"
	"encoding/json"
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
		"src_callback_msg": {jsonObjectForSourceChainCallback},
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
	callbackData := iapd.GetCallbackData()
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

// GetUserDefinedCustomMessage returns the custom message provided in the packet data memo.
// Custom message is expected to be base64 encoded.
// If no custom message is specified, nil is returned.
func (iapd InterchainAccountPacketData) GetUserDefinedCustomMessage() []byte {
	callbackData := iapd.GetCallbackData()
	if callbackData == nil {
		return nil
	}

	callbackMsg, ok := callbackData["src_callback_msg"].(string)
	if !ok {
		return nil
	}

	// base64 decode the callback message
	base64DecodedMsg, err := base64.StdEncoding.DecodeString(callbackMsg)
	if err != nil {
		return nil
	}

	return base64DecodedMsg
}

// UserDefinedGasLimit returns 0 (no-op). The gas limit of the executing
// transaction will be used.
func (iapd InterchainAccountPacketData) UserDefinedGasLimit() uint64 {
	return 0
}

func (iapd InterchainAccountPacketData) GetCallbackData() map[string]interface{} {
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
