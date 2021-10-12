package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

// msg types
const (
	TypeMsgRegisterCounterpartyAddress = "registerCounterpartyAddress"
	TypePayPacketFee                   = "payPacketFee"
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

// NewMsgPayPacketFee creates a new instance of MsgPayPacketFee
func NewMsgPayPacketFee(fee *Fee, sourcePortId, sourceChannelId, refundAccount string, relayers []string) *MsgPayPacketFee {
	return &MsgPayPacketFee{
		Fee:             fee,
		SourcePortId:    sourcePortId,
		SourceChannelId: sourceChannelId,
		RefundAccount:   refundAccount,
		Relayers:        relayers,
	}
}

// ValidateBasic performs a basic check of the MsgPayPacketFee fields
func (msg MsgPayPacketFee) ValidateBasic() error {
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgPayPacketFee) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.RefundAccount)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// NewMsgPayPacketAsync creates a new instance of MsgPayPacketFee
func NewMsgPayPacketFeeAsync(fee *Fee, packetId *channeltypes.PacketId, refundAccount string, relayers []string) *MsgPayPacketFeeAsync {
	return &MsgPayPacketFeeAsync{
		Fee:           fee,
		PacketId:      packetId,
		RefundAccount: refundAccount,
		Relayers:      relayers,
	}
}

// ValidateBasic performs a basic check of the MsgPayPacketFeeAsync fields
func (msg MsgPayPacketFeeAsync) ValidateBasic() error {
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgPayPacketFeeAsync) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.RefundAccount)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}
