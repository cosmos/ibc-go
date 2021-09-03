package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// msg types
const (
	TypeMsgRegisterCounterPartyAddress = "registerCounterPartyAddress"
)

// NewMsgRegisterCounterPartyAddress
func NewMsgRegisterCounterpartyAddress(sourceAddress, counterpartyAddress string) *MsgRegisterCounterpartyAddress {
	return &MsgRegisterCounterpartyAddress{Address: sourceAddress, CounterpartyAddress: counterpartyAddress}
}

// ValidateBasic performs a basic check of the Msg fields
func (msg MsgRegisterCounterpartyAddress) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrap(err, "Incorrect source relayer address")
	}

	_, err = sdk.AccAddressFromBech32(msg.CounterpartyAddress)
	if err != nil {
		return sdkerrors.Wrap(err, "Incorrect counterparty relayer address")
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgRegisterCounterpartyAddress) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
