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

	// if any of the fee's are invalid return an error
	if !msg.IdentifiedPacketFee.Fee.AckFee.IsValid() || !msg.IdentifiedPacketFee.Fee.ReceiveFee.IsValid() || !msg.IdentifiedPacketFee.Fee.TimeoutFee.IsValid() {
		return sdkerrors.ErrInvalidCoins
	}

	// if all three fee's are zero or empty return an error
	if msg.IdentifiedPacketFee.Fee.AckFee.IsZero() && msg.IdentifiedPacketFee.Fee.ReceiveFee.IsZero() && msg.IdentifiedPacketFee.Fee.TimeoutFee.IsZero() {
		return sdkerrors.ErrInvalidCoins
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

func NewIdentifiedPacketFee(packetId *channeltypes.PacketId, fee Fee, refundAcc string, relayers []string) *IdentifiedPacketFee {
	return &IdentifiedPacketFee{
		PacketId:  packetId,
		Fee:       fee,
		RefundAcc: refundAcc,
		Relayers:  relayers,
	}
}

func NewPacketId(channelId string, id uint64) *channeltypes.PacketId {
	return &channeltypes.PacketId{ChannelId: channelId, PortId: PortKey, Sequence: id}
}
