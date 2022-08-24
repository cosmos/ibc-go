package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
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
func NewMsgSubmitTx(connectionID, owner string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, msgs []*codectypes.Any) *MsgSubmitTx {
	return &MsgSubmitTx{
		ConnectionId:     connectionID,
		Owner:            owner,
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
		Msg:              msgs,
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
