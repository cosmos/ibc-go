package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgPushNewWasmCode{}
)

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

