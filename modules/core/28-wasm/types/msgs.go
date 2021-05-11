package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

var _ sdk.Msg = &MsgPushNewWASMCode{}

func (m *MsgPushNewWASMCode) Route() string {
	return host.RouterKey
}

func (m *MsgPushNewWASMCode) Type() string {
	return "wasm_push_new_code"
}

func (m *MsgPushNewWASMCode) ValidateBasic() error {
	if len(m.ClientType) == 0 {
		return sdkerrors.Wrapf(ErrEmptyClientType,
			"empty client type",
		)
	}

	if len(m.Code) == 0 {
		return sdkerrors.Wrapf(ErrEmptyWASMCode,
			"empty wasm code",
		)
	}

	return nil
}

func (m *MsgPushNewWASMCode) GetSignBytes() []byte {
	panic("IBC messages do not support amino")
}

func (m *MsgPushNewWASMCode) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
