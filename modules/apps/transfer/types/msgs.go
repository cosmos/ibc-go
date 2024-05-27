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
	MaximumTokensLength   = 100   // maximum number of tokens that can be transferred in a single message (value chosen arbitrarily)
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
	tokens sdk.Coins, sender, receiver string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64,
	memo string,
	forwardingPath *ForwardingInfo,
) *MsgTransfer {
	return &MsgTransfer{
		SourcePort:       sourcePort,
		SourceChannel:    sourceChannel,
		Sender:           sender,
		Receiver:         receiver,
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
		Memo:             memo,
		Tokens:           tokens,
		ForwardingPath:   forwardingPath,
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

	if len(msg.Tokens) == 0 && !isValidIBCCoin(msg.Token) {
		return errorsmod.Wrap(ibcerrors.ErrInvalidCoins, "either token or token array must be filled")
	}

	if len(msg.Tokens) != 0 && isValidIBCCoin(msg.Token) {
		return errorsmod.Wrap(ibcerrors.ErrInvalidCoins, "cannot fill both token and token array")
	}

	if len(msg.Tokens) > MaximumTokensLength {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidCoins, "number of tokens must not exceed %d", MaximumTokensLength)
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

	for _, coin := range msg.GetCoins() {
		if err := validateIBCCoin(coin); err != nil {
			return errorsmod.Wrapf(ibcerrors.ErrInvalidCoins, "%s: %s", err.Error(), coin.String())
		}
	}

	return nil
}

// GetCoins returns the tokens which will be transferred.
// If MsgTransfer is populated in the Token field, only that field
// will be returned in the coin array.
func (msg MsgTransfer) GetCoins() sdk.Coins {
	coins := msg.Tokens
	if isValidIBCCoin(msg.Token) {
		coins = []sdk.Coin{msg.Token}
	}
	return coins
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
