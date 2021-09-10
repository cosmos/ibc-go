package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// msg types
const (
	TypeMsgRegisterCounterpartyAddress = "registerCounterpartyAddress"
	TypeMsgEscrowPacketFee             = "escrowPacketFee"
)

// NewMsgRegisterCounterpartyAddress creates a new instance of MsgRegisterCounterpartyAddress
func NewMsgRegisterCounterpartyAddress(address, counterpartyAddress string) *MsgRegisterCounterpartyAddress {
	return &MsgRegisterCounterpartyAddress{
		Address:             address,
		CounterpartyAddress: counterpartyAddress,
	}
}

// ValidateBasic performs a basic check of the MsgRegisterCounterpartyAddress fields
func (msg MsgRegisterCounterpartyAddress) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to convert msg.Address into sdk.AccAddress")
	}

	_, err = sdk.AccAddressFromBech32(msg.CounterpartyAddress)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to convert msg.CounterpartyAddress into sdk.AccAddress")
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

// NewMsgEscrowPacketFee creates a new instance of MsgEscrowPacketFee
func NewMsgEscrowPacketFee(incentivizedPacket *IdentifiedPacketFee, relayers []string) *MsgEscrowPacketFee {
	return &MsgEscrowPacketFee{
		IncentivizedPacket: incentivizedPacket,
		Relayers:           relayers,
	}
}

// ValidateBasic performs a basic check of the MsgEscrowPacketFee fields
func (msg MsgEscrowPacketFee) ValidateBasic() error {
	//TODO
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgEscrowPacketFee) GetSigners() []sdk.AccAddress {
	//TODO
	return []sdk.AccAddress{}
}
