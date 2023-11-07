package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

var (
	_ sdk.Msg              = (*MsgStoreCode)(nil)
	_ sdk.Msg              = (*MsgMigrateContract)(nil)
	_ sdk.HasValidateBasic = (*MsgStoreCode)(nil)
	_ sdk.HasValidateBasic = (*MsgMigrateContract)(nil)
)

// MsgStoreCode creates a new MsgStoreCode instance
func NewMsgStoreCode(signer string, code []byte) *MsgStoreCode {
	return &MsgStoreCode{
		Signer:       signer,
		WasmByteCode: code,
	}
}

// ValidateBasic implements sdk.Msg
func (m MsgStoreCode) ValidateBasic() error {
	if err := ValidateWasmCode(m.WasmByteCode); err != nil {
		return err
	}

	_, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}

// NewMsgRemoveCodeHash creates a new MsgRemoveCodeHash instance
//
//nolint:interfacer
func NewMsgRemoveCodeHash(signer string, codeHash []byte) *MsgRemoveCodeHash {
	return &MsgRemoveCodeHash{
		Signer:   signer,
		CodeHash: codeHash,
	}
}

// ValidateBasic implements sdk.Msg
func (m MsgRemoveCodeHash) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	if err := ValidateWasmCodeHash(m.CodeHash); err != nil {
		return err
	}

	return nil
}

// MsgMigrateContract creates a new MsgMigrateContract instance
func NewMsgMigrateContract(signer, clientID string, codeHash, migrateMsg []byte) *MsgMigrateContract {
	return &MsgMigrateContract{
		Signer:   signer,
		ClientId: clientID,
		CodeHash: codeHash,
		Msg:      migrateMsg,
	}
}

// ValidateBasic implements sdk.Msg
func (m MsgMigrateContract) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	if err := ValidateWasmCodeHash(m.CodeHash); err != nil {
		return err
	}

	if err := ValidateClientID(m.ClientId); err != nil {
		return err
	}

	if len(m.Msg) == 0 {
		return errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "migrate message cannot be empty")
	}

	return nil
}
