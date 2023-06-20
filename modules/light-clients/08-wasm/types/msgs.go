package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var TypeMsgPushNewWasmCode = "push_wasm_code"
var _ sdk.Msg = &MsgPushNewWasmCode{}

// NewMsgPushNewWasmCode creates a new MsgPushNewWasmCode instance
//
//nolint:interfacer

// Route Implements Msg.
func (msg MsgPushNewWasmCode) Route() string { return ModuleName }

// Type Implements Msg.
func (msg MsgPushNewWasmCode) Type() string { return TypeMsgPushNewWasmCode }

func NewMsgPushNewWasmCode(signer string, code []byte) *MsgPushNewWasmCode {
	return &MsgPushNewWasmCode{
		Signer: signer,
		Code:   code,
	}
}

func (m MsgPushNewWasmCode) ValidateBasic() error {
	if len(m.Code) == 0 {
		return sdkerrors.Wrapf(ErrWasmEmptyCode,
			"empty wasm code",
		)
	}

	return nil
}

func (m MsgPushNewWasmCode) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
