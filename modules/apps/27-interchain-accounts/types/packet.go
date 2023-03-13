package types

import (
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

var _ exported.CallbackPacketDataI = (*InterchainAccountPacketData)(nil)

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

/**

ADR-8 CallbackPacketData implementation

InterchainAccountPacketData implements CallbackPacketDataI interface. This will allow middlewares targetting specific VMs
to retrieve the desired callback addresses for the ICA packet on the source and destination chains.

The Memo is used to set the desired callback addresses.

The Memo format is defined like so:

```json
{
	// ... other memo fields we don't care about
	"callbacks": {
		"src_callback_address": {contractAddrOnSrcChain},
		"dest_callback_address": {contractAddrOnDestChain},
		"src_callback_msg": {jsonObjectForSrcChainCallback},
		"dest_callback_msg": {jsonObjectForDestChainCallback},
	}

}
```

**/

// ADR-8 middleware should callback on the sender address on the source chain
// if the sender address is an IBC Actor (i.e. smart contract that accepts IBC callbacks)
func (iapd InterchainAccountPacketData) GetSrcCallbackAddress() (addr string) {
	if len(iapd.Memo) == 0 {
		return
	}

	jsonObject := make(map[string]interface{})
	// the jsonObject must be a valid JSON object
	err := json.Unmarshal([]byte(iapd.Memo), &jsonObject)
	if err != nil {
		return
	}

	callbackData, ok := jsonObject["callbacks"].(map[string]interface{})
	if !ok {
		return
	}

	callbackAddr := callbackData["src_callback_address"].(string)
	return callbackAddr
}

// ADR-8 middleware should callback on the receiver address on the destination chain
// if the receiver address is an IBC Actor (i.e. smart contract that accepts IBC callbacks)
func (iapd InterchainAccountPacketData) GetDestCallbackAddress() (addr string) {
	if len(iapd.Memo) == 0 {
		return
	}

	jsonObject := make(map[string]interface{})
	// the jsonObject must be a valid JSON object
	err := json.Unmarshal([]byte(iapd.Memo), &jsonObject)
	if err != nil {
		return
	}

	callbackData, ok := jsonObject["callbacks"].(map[string]interface{})
	if !ok {
		return
	}

	callbackAddr := callbackData["dest_callback_address"].(string)
	return callbackAddr
}

// no-op on this method to use relayer passed in gas
func (fptd InterchainAccountPacketData) UserDefinedGasLimit() uint64 {
	return 0
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
