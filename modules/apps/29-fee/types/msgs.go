package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
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
	err := host.ChannelIdentifierValidator(msg.SourceChannelId)
	if err != nil {
		return err
	}

	// validate portId
	err = host.PortIdentifierValidator(msg.SourcePortId)
	if err != nil {
		return err
	}

	// signer check
	_, err = sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to convert msg.Signer into sdk.AccAddress")
	}

	// enforce relayer is nil
	if msg.Relayers != nil {
		return ErrRelayersNotNil
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
func NewMsgPayPacketFeeAsync(identifiedPacketFee IdentifiedPacketFee, signer string) *MsgPayPacketFeeAsync {
	return &MsgPayPacketFeeAsync{
		IdentifiedPacketFee: identifiedPacketFee,
		Signer:              signer,
	}
}

// ValidateBasic performs a basic check of the MsgPayPacketFeeAsync fields
func (msg MsgPayPacketFeeAsync) ValidateBasic() error {
	// validate channelId
	err := host.ChannelIdentifierValidator(msg.IdentifiedPacketFee.PacketId.ChannelId)
	if err != nil {
		return err
	}

	// validate portId
	err = host.PortIdentifierValidator(msg.IdentifiedPacketFee.PacketId.PortId)
	if err != nil {
		return err
	}

	// signer check
	_, err = sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to convert msg.Signer into sdk.AccAddress")
	}

	// enforce relayer is nil
	if msg.IdentifiedPacketFee.Relayers != nil {
		return ErrRelayersNotNil
	}

	// ensure sequence is not 0
	if msg.IdentifiedPacketFee.PacketId.Sequence == 0 {
		return sdkerrors.ErrInvalidSequence
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgPayPacketFeeAsync) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func NewIdentifiedPacketFee(packetId *channeltypes.PacketId, fee Fee, relayers []string) *IdentifiedPacketFee {
	return &IdentifiedPacketFee{
		PacketId: packetId,
		Fee:      fee,
		Relayers: relayers,
	}
}
