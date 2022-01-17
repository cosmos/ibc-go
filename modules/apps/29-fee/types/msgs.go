package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// msg types
const (
	TypeMsgRegisterCounterpartyAddress = "registerCounterpartyAddress"
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

	if msg.CounterpartyAddress == "" {
		return ErrCounterpartyAddressEmpty
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
func NewMsgPayPacketFee(fee Fee, sourcePortId, sourceChannelId, signer string, relayers []string) *MsgPayPacketFee {
	return &MsgPayPacketFee{
		Fee:             fee,
		SourcePortId:    sourcePortId,
		SourceChannelId: sourceChannelId,
		Signer:          signer,
		Relayers:        relayers,
	}
}

// ValidateBasic performs a basic check of the MsgPayPacketFee fields
func (msg MsgPayPacketFee) ValidateBasic() error {
	// validate channelId
	if err := host.ChannelIdentifierValidator(msg.SourceChannelId); err != nil {
		return err
	}

	// validate portId
	if err := host.PortIdentifierValidator(msg.SourcePortId); err != nil {
		return err
	}

	// signer check
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to convert msg.Signer into sdk.AccAddress")
	}

	// enforce relayer is nil
	if msg.Relayers != nil {
		return ErrRelayersNotNil
	}

	// if any of the fee's are invalid return an error
	if !msg.Fee.AckFee.IsValid() || !msg.Fee.ReceiveFee.IsValid() || !msg.Fee.TimeoutFee.IsValid() {
		return sdkerrors.ErrInvalidCoins
	}

	// if all three fee's are zero or empty return an error
	if msg.Fee.AckFee.IsZero() && msg.Fee.ReceiveFee.IsZero() && msg.Fee.TimeoutFee.IsZero() {
		return sdkerrors.ErrInvalidCoins
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgPayPacketFee) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// NewMsgPayPacketAsync creates a new instance of MsgPayPacketFee
func NewMsgPayPacketFeeAsync(identifiedPacketFee IdentifiedPacketFee) *MsgPayPacketFeeAsync {
	return &MsgPayPacketFeeAsync{
		IdentifiedPacketFee: identifiedPacketFee,
	}
}

// ValidateBasic performs a basic check of the MsgPayPacketFeeAsync fields
func (msg MsgPayPacketFeeAsync) ValidateBasic() error {
	// signer check
	_, err := sdk.AccAddressFromBech32(msg.IdentifiedPacketFee.RefundAddress)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to convert msg.Signer into sdk.AccAddress")
	}

	if err = msg.IdentifiedPacketFee.Validate(); err != nil {
		return sdkerrors.Wrap(err, "Invalid IdentifiedPacketFee")
	}

	return nil
}

// GetSigners implements sdk.Msg
// The signer of the fee message must be the refund address
func (msg MsgPayPacketFeeAsync) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.IdentifiedPacketFee.RefundAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func NewIdentifiedPacketFee(packetId *channeltypes.PacketId, fee Fee, refundAddr string, relayers []string) *IdentifiedPacketFee {
	return &IdentifiedPacketFee{
		PacketId:      packetId,
		Fee:           fee,
		RefundAddress: refundAddr,
		Relayers:      relayers,
	}
}

func (fee IdentifiedPacketFee) Validate() error {
	// validate PacketId
	if err := fee.PacketId.Validate(); err != nil {
		return sdkerrors.Wrap(err, "Invalid PacketId")
	}

	_, err := sdk.AccAddressFromBech32(fee.RefundAddress)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to convert RefundAddress into sdk.AccAddress")
	}

	// if any of the fee's are invalid return an error
	if !fee.Fee.AckFee.IsValid() || !fee.Fee.ReceiveFee.IsValid() || !fee.Fee.TimeoutFee.IsValid() {
		return sdkerrors.ErrInvalidCoins
	}

	// if all three fee's are zero or empty return an error
	if fee.Fee.AckFee.IsZero() && fee.Fee.ReceiveFee.IsZero() && fee.Fee.TimeoutFee.IsZero() {
		return sdkerrors.ErrInvalidCoins
	}

	// enforce relayer is nil
	if fee.Relayers != nil {
		return ErrRelayersNotNil
	}

	return nil
}
