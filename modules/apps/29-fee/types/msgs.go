package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v4/modules/core/24-host"
)

// msg types
const (
	TypeMsgPayPacketFee      = "payPacketFee"
	TypeMsgPayPacketFeeAsync = "payPacketFeeAsync"
)

// NewMsgRegisterPayee creates a new instance of MsgRegisterPayee
func NewMsgRegisterPayee(portID, channelID, relayerAddr, payeeAddr string) *MsgRegisterPayee {
	return &MsgRegisterPayee{
		PortId:    portID,
		ChannelId: channelID,
		Relayer:   relayerAddr,
		Payee:     payeeAddr,
	}
}

// ValidateBasic implements sdk.Msg and performs basic stateless validation
func (msg MsgRegisterPayee) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return err
	}

	if err := host.ChannelIdentifierValidator(msg.ChannelId); err != nil {
		return err
	}

	if msg.Relayer == msg.Payee {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "relayer address and payee must not be equal")
	}

	_, err := sdk.AccAddressFromBech32(msg.Relayer)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to create sdk.AccAddress from relayer address")
	}

	_, err = sdk.AccAddressFromBech32(msg.Payee)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to create sdk.AccAddress from payee address")
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgRegisterPayee) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Relayer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

// NewMsgRegisterCounterpartyPayee creates a new instance of MsgRegisterCounterpartyPayee
func NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr string) *MsgRegisterCounterpartyPayee {
	return &MsgRegisterCounterpartyPayee{
		PortId:            portID,
		ChannelId:         channelID,
		Relayer:           relayerAddr,
		CounterpartyPayee: counterpartyPayeeAddr,
	}
}

// ValidateBasic performs a basic check of the MsgRegisterCounterpartyAddress fields
func (msg MsgRegisterCounterpartyPayee) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return err
	}

	if err := host.ChannelIdentifierValidator(msg.ChannelId); err != nil {
		return err
	}

	_, err := sdk.AccAddressFromBech32(msg.Relayer)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to create sdk.AccAddress from relayer address")
	}

	if strings.TrimSpace(msg.CounterpartyPayee) == "" {
		return ErrCounterpartyPayeeEmpty
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgRegisterCounterpartyPayee) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Relayer)
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
func NewMsgPayPacketFeeAsync(packetID channeltypes.PacketId, packetFee PacketFee) *MsgPayPacketFeeAsync {
	return &MsgPayPacketFeeAsync{
		PacketId:  packetID,
		PacketFee: packetFee,
	}
}

// ValidateBasic performs a basic check of the MsgPayPacketFeeAsync fields
func (msg MsgPayPacketFeeAsync) ValidateBasic() error {
	if err := msg.PacketId.Validate(); err != nil {
		return err
	}

	if err := msg.PacketFee.Validate(); err != nil {
		return err
	}

	return nil
}

// GetSigners implements sdk.Msg
// The signer of the fee message must be the refund address
func (msg MsgPayPacketFeeAsync) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.PacketFee.RefundAddress)
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
