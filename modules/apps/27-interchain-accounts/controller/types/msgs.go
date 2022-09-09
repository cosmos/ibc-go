package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	channelerrors "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

var _ sdk.Msg = &MsgRegisterAccount{}

// NewMsgRegisterAccount creates a new instance of MsgRegisterAccount
func NewMsgRegisterAccount(connectionID, owner, version string) *MsgRegisterAccount {
	return &MsgRegisterAccount{
		ConnectionId: connectionID,
		Owner:        owner,
		Version:      version,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgRegisterAccount) ValidateBasic() error {
	if err := host.ConnectionIdentifierValidator(msg.ConnectionId); err != nil {
		return sdkerrors.Wrap(err, "invalid connection ID")
	}

	if strings.TrimSpace(msg.Owner) == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "owner address cannot be empty")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "failed to parse owner address: %s", msg.Owner)
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgRegisterAccount) GetSigners() []sdk.AccAddress {
	accAddr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{accAddr}
}

// NewMsgSendTx creates a new instance of MsgSendTx
func NewMsgSendTx(owner, connectionID string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, packetData icatypes.InterchainAccountPacketData) *MsgSendTx {
	return &MsgSendTx{
		ConnectionId:     connectionID,
		Owner:            owner,
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
		PacketData:       packetData,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgSendTx) ValidateBasic() error {
	if err := host.ConnectionIdentifierValidator(msg.ConnectionId); err != nil {
		return sdkerrors.Wrap(err, "invalid connection ID")
	}

	if strings.TrimSpace(msg.Owner) == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "owner address cannot be empty")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "failed to parse owner address: %s", msg.Owner)
	}

	if msg.TimeoutHeight.IsZero() && msg.TimeoutTimestamp == 0 {
		return sdkerrors.Wrap(channelerrors.ErrInvalidTimeout, "msg timeout height and msg timeout timestamp cannot both be 0")
	}

	if err := msg.PacketData.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "invalid interchain account packet data")
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgSendTx) GetSigners() []sdk.AccAddress {
	accAddr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{accAddr}
}
