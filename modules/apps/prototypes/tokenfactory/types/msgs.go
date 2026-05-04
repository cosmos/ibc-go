package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgCreateDenom{}
	_ sdk.Msg = &MsgMint{}
	_ sdk.Msg = &MsgBurn{}
	_ sdk.Msg = &MsgChangeAdmin{}
	_ sdk.Msg = &MsgRenounceAdmin{}
)

// NewMsgCreateDenom creates a new MsgCreateDenom instance
func NewMsgCreateDenom(sender, denom string) *MsgCreateDenom {
	return &MsgCreateDenom{
		Sender: sender,
		Denom:  denom,
	}
}

// ValidateBasic implements Msg.ValidateBasic
func (msg MsgCreateDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidCreator, "error: %s", err.Error())
	}

	return ValidateTokenFactoryDenom(msg.Denom)
}

// NewMsgMint creates a new MsgMint instance
func NewMsgMint(from, address string, amount sdk.Coin) *MsgMint {
	return &MsgMint{
		From:    from,
		Address: address,
		Amount:  amount,
	}
}

// ValidateBasic implements Msg.ValidateBasic
func (msg MsgMint) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.From); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "from: %s, error: %s", msg.From, err.Error())
	}

	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "address: %s, error: %s", msg.Address, err.Error())
	}

	if !msg.Amount.IsValid() || msg.Amount.IsZero() {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount: %s", msg.Amount.String())
	}

	return ValidateTokenFactoryDenom(msg.Amount.Denom)
}

// NewMsgBurn creates a new MsgBurn instance
func NewMsgBurn(from string, amount sdk.Coin) *MsgBurn {
	return &MsgBurn{
		From:   from,
		Amount: amount,
	}
}

// ValidateBasic implements Msg.ValidateBasic
func (msg MsgBurn) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.From); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "from: %s, error: %s", msg.From, err.Error())
	}

	if !msg.Amount.IsValid() || msg.Amount.IsZero() {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount: %s", msg.Amount.String())
	}

	return ValidateTokenFactoryDenom(msg.Amount.Denom)
}

// NewMsgChangeAdmin creates a new MsgChangeAdmin instance
func NewMsgChangeAdmin(sender, denom, newAdmin string) *MsgChangeAdmin {
	return &MsgChangeAdmin{
		Sender:   sender,
		Denom:    denom,
		NewAdmin: newAdmin,
	}
}

// ValidateBasic implements Msg.ValidateBasic
func (msg MsgChangeAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "sender: %s, error: %s", msg.Sender, err.Error())
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewAdmin); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "new_admin: %s, error: %s", msg.NewAdmin, err.Error())
	}

	return ValidateTokenFactoryDenom(msg.Denom)
}

// NewMsgRenounceAdmin creates a new MsgRenounceAdmin instance
func NewMsgRenounceAdmin(sender, denom string) *MsgRenounceAdmin {
	return &MsgRenounceAdmin{
		Sender: sender,
		Denom:  denom,
	}
}

// ValidateBasic implements Msg.ValidateBasic
func (msg MsgRenounceAdmin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "sender: %s, error: %s", msg.Sender, err.Error())
	}

	return ValidateTokenFactoryDenom(msg.Denom)
}
