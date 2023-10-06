package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg              = (*MsgStoreCode)(nil)
	_ sdk.HasValidateBasic = (*MsgStoreCode)(nil)
)

// MsgStoreCode creates a new MsgStoreCode instance
//
//nolint:interfacer
func NewMsgStoreCode(signer string, code []byte) *MsgStoreCode {
	return &MsgStoreCode{
		Signer:       signer,
		WasmByteCode: code,
	}
}

// ValidateBasic implements sdk.Msg
func (m MsgStoreCode) ValidateBasic() error {
	if len(m.WasmByteCode) == 0 {
		return ErrWasmEmptyCode
	}

	return nil
}

// GetSigners implements sdk.Msg
func (m MsgStoreCode) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
