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

// NewMsgRegisterPayee creates a new instance of MsgRegisterPayee
func NewMsgRegisterPayee(portID, channelID, relayerAddr, payeeAddr string) *MsgRegisterPayee {
	return &MsgRegisterPayee{
		RelayerAddress: relayerAddr,
		Payee:          payeeAddr,
		PortId:         portID,
		ChannelId:      channelID,
	}
}

// ValidateBasic implements sdk.Msg and performs basic stateless validation
func (msg MsgRegisterPayee) ValidateBasic() error {
	if msg.RelayerAddress == msg.Payee {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "relayer address and payee must not be equal")
	}

	_, err := sdk.AccAddressFromBech32(msg.RelayerAddress)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to create sdk.AccAddress from relayer address")
	}

	_, err = sdk.AccAddressFromBech32(msg.Payee)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to create sdk.AccAddress from payee address")
	}

	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return err
	}

	if err := host.ChannelIdentifierValidator(msg.ChannelId); err != nil {
		return err
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgRegisterPayee) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.RelayerAddress)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

// NewMsgRegisterCounterpartyAddress creates a new instance of MsgRegisterCounterpartyAddress
func NewMsgRegisterCounterpartyAddress(portID, channelID, address, counterpartyAddress string) *MsgRegisterCounterpartyAddress {
	return &MsgRegisterCounterpartyAddress{
		Address:             address,
		CounterpartyAddress: counterpartyAddress,
		PortId:              portID,
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

	// validate portId
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return err
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
