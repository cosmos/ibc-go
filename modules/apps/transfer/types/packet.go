package types

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

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

var _ exported.CallbackPacketData = (*FungibleTokenPacketData)(nil)

// NewFungibleTokenPacketData contructs a new FungibleTokenPacketData instance
func NewFungibleTokenPacketData(
	denom string, amount string,
	sender, receiver string,
	memo string,
) FungibleTokenPacketData {
	return FungibleTokenPacketData{
		Denom:    denom,
		Amount:   amount,
		Sender:   sender,
		Receiver: receiver,
		Memo:     memo,
	}
}

// ValidateBasic is used for validating the token transfer.
// NOTE: The addresses formats are not validated as the sender and recipient can have different
// formats defined by their corresponding chains that are not known to IBC.
func (ftpd FungibleTokenPacketData) ValidateBasic() error {
	amount, ok := sdkmath.NewIntFromString(ftpd.Amount)
	if !ok {
		return errorsmod.Wrapf(ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", ftpd.Amount)
	}
	if !amount.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount must be strictly positive: got %d", amount)
	}
	if strings.TrimSpace(ftpd.Sender) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "sender address cannot be blank")
	}
	if strings.TrimSpace(ftpd.Receiver) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "receiver address cannot be blank")
	}
	return ValidatePrefixedDenom(ftpd.Denom)
}

// GetBytes is a helper for serialising
func (ftpd FungibleTokenPacketData) GetBytes() []byte {
	return sdk.MustSortJSON(mustProtoMarshalJSON(&ftpd))
}

/*

ADR-8 CallbackPacketData implementation

FungibleTokenPacketData implements CallbackPacketData interface. This will allow middlewares targeting specific VMs
to retrieve the desired callback addresses for the ICS20 packet on the source and destination chains.

The Memo is used to ensure that the callback is desired by the user. This allows a user to send an ICS20 packet
to a contract with ADR-8 enabled without automatically triggering the callback logic which may lead to unexpected
behavior.

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

For transfer, we will NOT enforce that the source callback address is the same as sender and destination callback address is the same as receiver.

*/

// GetSourceCallbackAddress returns the source callback address
// if it is specified in the packet data memo.
// If no callback address is specified or the memo is improperly formatted, an empty string is returned.
//
// The memo is expected to contain the source callback address in the following format:
// { "src_callback": { "address": {stringCallbackAddress}}
//
// ADR-8 middleware should callback on the returned address if it is a PacketActor
// (i.e. smart contract that accepts IBC callbacks).
func (ftpd FungibleTokenPacketData) GetSourceCallbackAddress() string {
	return ftpd.getCallbackAddress("src_callback")
}

// GetDestCallbackAddress returns the destination callback address
// if it is specified in the packet data memo.
// If no callback address is specified or the memo is improperly formatted, an empty string is returned.
//
// The memo is expected to contain the destination callback address in the following format:
// { "dest_callback": { "address": {stringCallbackAddress}}
//
// ADR-8 middleware should callback on the returned address if it is a PacketActor
// (i.e. smart contract that accepts IBC callbacks).
func (ftpd FungibleTokenPacketData) GetDestCallbackAddress() string {
	return ftpd.getCallbackAddress("dest_callback")
}

// GetSourceUserDefinedGasLimit returns the custom gas limit provided for source callbacks
// if it is specified in the packet data memo.
// If no gas limit is specified or the gas limit is improperly formatted, 0 is returned.
//
// The memo is expected to specify the user defined gas limit in the following format:
// { "src_callback": { ... , "gas_limit": {stringForCallback} }
//
// Note: the user defined gas limit must be set as a string and not a json number.
func (ftpd FungibleTokenPacketData) GetSourceUserDefinedGasLimit() uint64 {
	return ftpd.getUserDefinedGasLimit("src_callback")
}

// GetDestUserDefinedGasLimit returns the custom gas limit provided for destination callbacks
// if it is specified in the packet data memo.
// If no gas limit is specified or the gas limit is improperly formatted, 0 is returned.
//
// The memo is expected to specify the user defined gas limit in the following format:
// { "dest_callback": { ... , "gas_limit": {stringForCallback} }
//
// Note: the user defined gas limit must be set as a string and not a json number.
func (ftpd FungibleTokenPacketData) GetDestUserDefinedGasLimit() uint64 {
	return ftpd.getUserDefinedGasLimit("dest_callback")
}

// GetPacketSender returns the sender address of the packet.
func (ftpd FungibleTokenPacketData) GetPacketSender(srcPortID string) string {
	return ftpd.Sender
}

// getUserDefinedGasLimit returns the custom gas limit provided for callbacks
// if it is specified in the packet data memo.
// If no gas limit is specified or the gas limit is improperly formatted, 0 is returned.
//
// The memo is expected to specify the user defined gas limit in the following format:
// { "{callbackKey}": { ... , "gas_limit": {stringForCallback} }
//
// Note: the user defined gas limit must be set as a string and not a json number.
func (ftpd FungibleTokenPacketData) getUserDefinedGasLimit(callbackKey string) uint64 {
	callbackData := ftpd.getCallbackData(callbackKey)
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

// getCallbackAddress returns the callback address if it is specified in the packet data memo.
// If no callback address is specified or the memo is improperly formatted, an empty string is returned.
//
// The memo is expected to contain the callback address in the following format:
// { "{callbackKey}": { "address": {stringCallbackAddress}}
//
// ADR-8 middleware should callback on the returned address if it is a PacketActor
// (i.e. smart contract that accepts IBC callbacks).
func (ftpd FungibleTokenPacketData) getCallbackAddress(callbackKey string) string {
	callbackData := ftpd.getCallbackData(callbackKey)
	if callbackData == nil {
		return ""
	}

	callbackAddress, ok := callbackData["address"].(string)
	if !ok {
		return ""
	}

	return callbackAddress
}

// getCallbackData returns the memo as `map[string]interface{}` so that it can be
// interpreted as a json object with keys.
func (ftpd FungibleTokenPacketData) getCallbackData(callbackKey string) map[string]interface{} {
	if len(ftpd.Memo) == 0 {
		return nil
	}

	jsonObject := make(map[string]interface{})
	err := json.Unmarshal([]byte(ftpd.Memo), &jsonObject)
	if err != nil {
		return nil
	}

	callbackData, ok := jsonObject[callbackKey].(map[string]interface{})
	if !ok {
		return nil
	}

	return callbackData
}
