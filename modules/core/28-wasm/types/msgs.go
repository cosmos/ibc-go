package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

var _ sdk.Msg = &MsgPushNewWasmCode{}

func (m *MsgPushNewWasmCode) Route() string {
	return host.RouterKey
}

func (m *MsgPushNewWasmCode) Type() string {
	return "wasm_push_new_code"
}

func (m *MsgPushNewWasmCode) ValidateBasic() error {
	if len(m.Code) == 0 {
		return sdkerrors.Wrapf(ErrWasmEmptyCode,
			"empty wasm code",
		)
	}

	return nil
}

func (m *MsgPushNewWasmCode) GetSignBytes() []byte {
	panic("IBC messages do not support amino")
}

func (m *MsgPushNewWasmCode) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
