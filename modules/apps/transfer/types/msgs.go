package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacytx "github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
)

// msg types
const (
	TypeMsgTransfer = "transfer"
)

var (
	_ sdk.Msg            = (*MsgUpdateParams)(nil)
	_ sdk.Msg            = (*MsgTransfer)(nil)
	_ legacytx.LegacyMsg = (*MsgTransfer)(nil)
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

// GetSigners implements sdk.Msg
func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress {
	accAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{accAddr}
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

// Type implements legacytx.LegacyMsg
func (MsgTransfer) Type() string {
	return TypeMsgTransfer
}

// Route implements legacytx.LegacyMsg
func (MsgTransfer) Route() string {
	return RouterKey
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
	if !msg.Token.IsValid() {
		return errorsmod.Wrap(ibcerrors.ErrInvalidCoins, msg.Token.String())
	}
	if !msg.Token.IsPositive() {
		return errorsmod.Wrap(ibcerrors.ErrInsufficientFunds, msg.Token.String())
	}
	// NOTE: sender format must be validated as it is required by the GetSigners function.
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	if strings.TrimSpace(msg.Receiver) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "missing recipient address")
	}
	return ValidateIBCDenom(msg.Token.Denom)
}

// GetSignBytes implements legacytx.LegacyMsg
func (msg MsgTransfer) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(&msg))
}

// GetSigners implements sdk.Msg
func (msg MsgTransfer) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
