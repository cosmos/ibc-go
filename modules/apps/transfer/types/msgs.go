package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

const (
	MaximumReceiverLength = 2048  // maximum length of the receiver address in bytes (value chosen arbitrarily)
	MaximumMemoLength     = 32768 // maximum length of the memo in bytes (value chosen arbitrarily)
)

var (
	_ sdk.Msg              = (*MsgUpdateParams)(nil)
	_ sdk.Msg              = (*MsgTransfer)(nil)
	_ sdk.HasValidateBasic = (*MsgUpdateParams)(nil)
	_ sdk.HasValidateBasic = (*MsgTransfer)(nil)
)

// NewMsgUpdateParams creates a new MsgUpdateParams instance
func NewMsgUpdateParams(signer string, params Params) *MsgUpdateParams {
	return &MsgUpdateParams{
		Signer: signer,
		Params: params,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgUpdateParams) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}

// NewMsgTransfer creates a new MsgTransfer instance
func NewMsgTransfer(
	sourcePort, sourceChannel string,
	token sdk.Coin, sender, receiver string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64,
	memo string,
) *MsgTransfer {
	return &MsgTransfer{
		SourcePort:       sourcePort,
		SourceChannel:    sourceChannel,
		Token:            token,
		Sender:           sender,
		Receiver:         receiver,
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
		Memo:             memo,
	}
}

// ValidateBasic performs a basic check of the MsgTransfer fields.
// NOTE: timeout height or timestamp values can be 0 to disable the timeout.
// NOTE: The recipient addresses format is not validated as the format defined by
// the chain is not known to IBC.
func (msg MsgTransfer) ValidateBasic() error {
	if err := msg.validateIdentifiers(); err != nil {
		return err
	}

	if !isValidIBCCoin(msg.Token) {
		return errorsmod.Wrap(ibcerrors.ErrInvalidCoins, msg.Token.String())
	}

	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	if strings.TrimSpace(msg.Receiver) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "missing recipient address")
	}
	if len(msg.Receiver) > MaximumReceiverLength {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "recipient address must not exceed %d bytes", MaximumReceiverLength)
	}
	if len(msg.Memo) > MaximumMemoLength {
		return errorsmod.Wrapf(ErrInvalidMemo, "memo must not exceed %d bytes", MaximumMemoLength)
	}

	return nil
}

// validateIdentifiers checks if the source port and channel identifiers are valid
func (msg MsgTransfer) validateIdentifiers() error {
	if err := host.PortIdentifierValidator(msg.SourcePort); err != nil {
		return errorsmod.Wrapf(err, "invalid source port ID %s", msg.SourcePort)
	}
	if err := host.ChannelIdentifierValidator(msg.SourceChannel); err != nil {
		return errorsmod.Wrapf(err, "invalid source channel ID %s", msg.SourceChannel)
	}

	return nil
}

// isValidIBCCoin returns true if the token provided is valid,
// and should be used to transfer tokens.
func isValidIBCCoin(coin sdk.Coin) bool {
	return validateIBCCoin(coin) == nil
}

// validateIBCCoin returns true if the token provided is valid,
// and should be used to transfer tokens. The token must
// have a positive amount.
func validateIBCCoin(coin sdk.Coin) error {
	if err := coin.Validate(); err != nil {
		return err
	}
	if !coin.IsPositive() {
		return errorsmod.Wrap(ErrInvalidAmount, "amount must be positive")
	}
	if err := validateIBCDenom(coin.GetDenom()); err != nil {
		return errorsmod.Wrap(ErrInvalidDenomForTransfer, err.Error())
	}

	return nil
}
