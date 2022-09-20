package types

import (
	"bytes"
	"crypto/sha256"
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSubmitWasmLightClient{}

func (m *MsgSubmitWasmLightClient) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", m.Signer)
	}
	return m.WasmLightClient.ValidateBasic()
}

func (m *MsgSubmitWasmLightClient) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (w *WasmLightClient) ValidateBasic() error {
	if len(w.Code) == 0 {
		return sdkerrors.Wrapf(ErrWasmEmptyCode,
			"empty wasm code",
		)
	}
	if len(w.CodeHash) == 0 {
		return sdkerrors.Wrapf(ErrWasmEmptyCodeHash,
			"empty wasm code hash",
		)
	}

	calcHash := sha256.Sum256(w.Code)
	if !bytes.Equal(w.CodeHash, calcHash[:]) {
		return sdkerrors.Wrapf(ErrWasmInvalidCodeID, "code hash doesn't match the code")
	}

	if w.Name == "" {
		// TODO: add more validation, correct errors
		return fmt.Errorf("name is empty")
	}
	if w.Repository == "" {
		// TODO: add more validation, correct errors
		return fmt.Errorf("repository is empty")
	}
	return nil
}
