package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"

	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

// msg types
const (
	TypeMsgQuery = "query"
)

// NewMsgQuery creates a new MsgQuery instance
//nolint:interfacer
func NewMsgQuery(
	sourcePort, sourceChannel string,
	requests []abci.RequestQuery,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64,
	signer string,
) *MsgQuery {
	return &MsgQuery{
		SourcePort:       sourcePort,
		SourceChannel:    sourceChannel,
		Requests:         requests,
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
		Signer:           signer,
	}
}

// Route implements sdk.Msg
func (MsgQuery) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (MsgQuery) Type() string {
	return TypeMsgQuery
}

// ValidateBasic performs a basic check of the MsgQuery fields.
// NOTE: timeout height or timestamp values can be 0 to disable the timeout.
func (msg MsgQuery) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.SourcePort); err != nil {
		return sdkerrors.Wrap(err, "invalid source port ID")
	}
	if err := host.ChannelIdentifierValidator(msg.SourceChannel); err != nil {
		return sdkerrors.Wrap(err, "invalid source channel ID")
	}
	if msg.Signer != "" {
		_, err := sdk.AccAddressFromBech32(msg.Signer)
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}
	}
	return nil
}

// GetSignBytes implements sdk.Msg.
func (msg MsgQuery) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&msg))
}

// GetSigners implements sdk.Msg
func (msg MsgQuery) GetSigners() []sdk.AccAddress {
	var signers []sdk.AccAddress
	if msg.Signer == "" {
		return signers
	}
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return append(signers, signer)
}
