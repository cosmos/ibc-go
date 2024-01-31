package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
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
	sourcePort, sourceChannel string, token sdk.Coin, sender, receiver string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64,
	memo string, tokens ...sdk.Coin,
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
		Tokens:           tokens,
	}
}

// ValidateBasic performs a basic check of the MsgTransfer fields.
// NOTE: timeout height or timestamp values can be 0 to disable the timeout.
// NOTE: The recipient addresses format is not validated as the format defined by
// the chain is not known to IBC.
func (msg MsgTransfer) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.SourcePort); err != nil {
		return errorsmod.Wrap(err, "invalid source port ID")
	}
	if err := host.ChannelIdentifierValidator(msg.SourceChannel); err != nil {
		return errorsmod.Wrap(err, "invalid source channel ID")
	}

	if len(msg.Tokens) == 0 && !isValidToken(msg.Token) {
		return errorsmod.Wrap(ErrInvalidAmount, "either token or token array must be filled")
	}

	if len(msg.Tokens) != 0 && isValidToken(msg.Token) {
		return errorsmod.Wrap(ErrInvalidAmount, "cannot fill both token and token array")
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

	for _, token := range msg.GetTokens() {
		if !token.IsValid() {
			return errorsmod.Wrap(ibcerrors.ErrInvalidCoins, token.String())
		}
		if !token.IsPositive() {
			return errorsmod.Wrap(ibcerrors.ErrInsufficientFunds, token.String())
		}
		if err := ValidateIBCDenom(token.Denom); err != nil {
			return errorsmod.Wrap(ibcerrors.ErrInvalidCoins, token.Denom)
		}
	}

	return nil
}

// GetTokens returns the tokens which will be transferred.
func (msg MsgTransfer) GetTokens() []sdk.Coin {
	tokensToValidate := msg.Tokens
	if isValidToken(msg.Token) {
		tokensToValidate = []sdk.Coin{msg.Token}
	}
	return tokensToValidate
}

// isValidToken returns true if the token provided is valid,
// and should be used to transfer tokens.
// this function is used in case the user constructs a sdk.Coin literal
// instead of using the construction function.
func isValidToken(coin sdk.Coin) bool {
	if coin.IsNil() {
		return false
	}

	if strings.TrimSpace(coin.Denom) == "" {
		return false
	}

	if coin.Amount.IsZero() {
		return false
	}

	if coin.Amount.IsNegative() {
		return false
	}

	return true
}
