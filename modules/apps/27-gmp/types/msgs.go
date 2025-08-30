package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

const (
	MaximumReceiverLength = 2048  // maximum length of the receiver address in bytes (value chosen arbitrarily)
	MaximumMemoLength     = 32768 // maximum length of the memo in bytes (value chosen arbitrarily)
	MaximumSaltLength     = 32    // maximum length of the salt in bytes (value chosen arbitrarily)
	MaximumPayloadLength  = 32768 // maximum length of the payload in bytes (value chosen arbitrarily)
)

var (
	_ sdk.Msg              = (*MsgSendCall)(nil)
	_ sdk.HasValidateBasic = (*MsgSendCall)(nil)
)

// NewMsgSendCall creates a new MsgSendCall instance
func NewMsgSendCall(sourceClient, sender, receiver string, payload, salt []byte, timeoutTimestamp uint64, encoding, memo string) *MsgSendCall {
	return &MsgSendCall{
		SourceClient:     sourceClient,
		Sender:           sender,
		Receiver:         receiver,
		Payload:          payload,
		Salt:             salt,
		Memo:             memo,
		TimeoutTimestamp: timeoutTimestamp,
		Encoding:         encoding,
	}
}

// ValidateBasic performs a basic check of the MsgSendCall fields.
// NOTE: The recipient addresses format is not validated as the format defined by
// the chain is not known to IBC.
func (msg MsgSendCall) ValidateBasic() error {
	if err := msg.validateIdentifiers(); err != nil {
		return err
	}

	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	// receiver is allowed to be empty
	if len(msg.Receiver) > MaximumReceiverLength {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "recipient address must not exceed %d bytes", MaximumReceiverLength)
	}
	if len(msg.Payload) > MaximumPayloadLength {
		return errorsmod.Wrapf(ErrInvalidPayload, "payload must not exceed %d bytes", MaximumPayloadLength)
	}
	if len(msg.Salt) > MaximumSaltLength {
		return errorsmod.Wrapf(ErrInvalidSalt, "salt must not exceed %d bytes", MaximumSaltLength)
	}
	if len(msg.Memo) > MaximumMemoLength {
		return errorsmod.Wrapf(ErrInvalidMemo, "memo must not exceed %d bytes", MaximumMemoLength)
	}
	if msg.TimeoutTimestamp == 0 {
		return errorsmod.Wrap(ErrInvalidTimeoutTimestamp, "timeout timestamp must be greater than 0")
	}
	return validateEncoding(msg.Encoding)
}

// validateIdentifiers checks if the IBC identifiers are valid
func (msg MsgSendCall) validateIdentifiers() error {
	if err := host.ClientIdentifierValidator(msg.SourceClient); err != nil {
		return errorsmod.Wrapf(err, "invalid source client ID %s", msg.SourceClient)
	}

	return nil
}

// validateEncoding checks if the encoding is valid
func validateEncoding(encoding string) error {
	switch encoding {
	case "", EncodingProtobuf, EncodingJSON, EncodingABI:
		return nil
	default:
		return errorsmod.Wrapf(ErrInvalidEncoding, "unsupported encoding format %s", encoding)
	}
}
