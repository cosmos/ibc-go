package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgRegisterAccount) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{}
}

// NewMsgSubmitTx creates a new instance of MsgSubmitTx
func NewMsgSubmitTx(connectionID, owner, timeout string, msg []*codectypes.Any) *MsgSubmitTx {
	return &MsgSubmitTx{
		ConnectionId: connectionID,
		Owner:        owner,
		Timeout:      timeout,
		Msg:          msg,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgSubmitTx) ValidateBasic() error {
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgSubmitTx) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{}
}
