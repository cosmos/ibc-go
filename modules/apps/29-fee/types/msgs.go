package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// msg types
const (
	TypeMsgPayPacketFee      = "payPacketFee"
	TypeMsgPayPacketFeeAsync = "payPacketFeeAsync"
)

// NewMsgRegisterCounterpartyAddress creates a new instance of MsgRegisterCounterpartyAddress
func NewMsgRegisterCounterpartyAddress(address, counterpartyAddress, channelID string) *MsgRegisterCounterpartyAddress {
	return &MsgRegisterCounterpartyAddress{
		Address:             address,
		CounterpartyAddress: counterpartyAddress,
		ChannelId:           channelID,
	}
}

// ValidateBasic performs a basic check of the MsgRegisterCounterpartyAddress fields
func (msg MsgRegisterCounterpartyAddress) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to convert msg.Address into sdk.AccAddress")
	}

	if strings.TrimSpace(msg.CounterpartyAddress) == "" {
		return ErrCounterpartyAddressEmpty
	}

	// validate channelId
	if err := host.ChannelIdentifierValidator(msg.ChannelId); err != nil {
		return err
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
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return sdkerrors.Wrap(err, "failed to convert msg.Signer into sdk.AccAddress")
	}

	// enforce relayer is nil
	if msg.Relayers != nil {
		return ErrRelayersNotNil
	}

	if err := msg.Fee.Validate(); err != nil {
		return err
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

// Route implements sdk.Msg
func (msg MsgPayPacketFee) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg MsgPayPacketFee) Type() string {
	return TypeMsgPayPacketFee
}

// GetSignBytes implements sdk.Msg.
func (msg MsgPayPacketFee) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&msg))
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

// Route implements sdk.Msg
func (msg MsgPayPacketFeeAsync) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg MsgPayPacketFeeAsync) Type() string {
	return TypeMsgPayPacketFeeAsync
}

// GetSignBytes implements sdk.Msg.
func (msg MsgPayPacketFeeAsync) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&msg))
}

func NewIdentifiedPacketFee(packetId channeltypes.PacketId, fee Fee, refundAddr string, relayers []string) IdentifiedPacketFee {
	return IdentifiedPacketFee{
		PacketId:      packetId,
		Fee:           fee,
		RefundAddress: refundAddr,
		Relayers:      relayers,
	}
}

// Validate performs a stateless check of the IdentifiedPacketFee fields
func (fee IdentifiedPacketFee) Validate() error {
	// validate PacketId
	if err := fee.PacketId.Validate(); err != nil {
		return sdkerrors.Wrap(err, "Invalid PacketId")
	}

	_, err := sdk.AccAddressFromBech32(fee.RefundAddress)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to convert RefundAddress into sdk.AccAddress")
	}

	// enforce relayer is nil
	if fee.Relayers != nil {
		return ErrRelayersNotNil
	}

	if err := fee.Fee.Validate(); err != nil {
		return err
	}

	return nil
}
