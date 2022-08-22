package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewMsgRegisterPayee creates a new instance of MsgRegisterPayee
func NewMsgRegisterPayee(connectionID, owner, version string) *MsgRegisterAccount {
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
