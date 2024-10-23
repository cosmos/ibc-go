package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypesv1 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var (
	_ sdk.Msg              = (*MsgCreateChannel)(nil)
	_ sdk.HasValidateBasic = (*MsgCreateChannel)(nil)

	_ sdk.Msg              = (*MsgRegisterCounterparty)(nil)
	_ sdk.HasValidateBasic = (*MsgRegisterCounterparty)(nil)

	_ sdk.Msg              = (*MsgSendPacket)(nil)
	_ sdk.HasValidateBasic = (*MsgSendPacket)(nil)

	_ sdk.Msg              = (*MsgRecvPacket)(nil)
	_ sdk.HasValidateBasic = (*MsgRecvPacket)(nil)

	_ sdk.Msg              = (*MsgTimeout)(nil)
	_ sdk.HasValidateBasic = (*MsgTimeout)(nil)

	_ sdk.Msg              = (*MsgAcknowledgement)(nil)
	_ sdk.HasValidateBasic = (*MsgAcknowledgement)(nil)
)

// NewMsgCreateChannel creates a new MsgCreateChannel instance
func NewMsgCreateChannel(clientID string, merklePathPrefix commitmenttypesv2.MerklePath, signer string) *MsgCreateChannel {
	return &MsgCreateChannel{
		Signer:           signer,
		ClientId:         clientID,
		MerklePathPrefix: merklePathPrefix,
	}
}

// ValidateBasic performs basic checks on a MsgCreateChannel.
func (msg *MsgCreateChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	if err := host.ClientIdentifierValidator(msg.ClientId); err != nil {
		return err
	}

	if err := msg.MerklePathPrefix.ValidateAsPrefix(); err != nil {
		return err
	}

	return nil
}

// NewMsgRegisterCounterparty creates a new MsgRegisterCounterparty instance
func NewMsgRegisterCounterparty(channelID, counterpartyChannelID string, signer string) *MsgRegisterCounterparty {
	return &MsgRegisterCounterparty{
		Signer:                signer,
		ChannelId:             channelID,
		CounterpartyChannelId: counterpartyChannelID,
	}
}

// ValidateBasic performs basic checks on a MsgRegisterCounterparty.
func (msg *MsgRegisterCounterparty) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	if err := host.ChannelIdentifierValidator(msg.ChannelId); err != nil {
		return err
	}

	if err := host.ChannelIdentifierValidator(msg.CounterpartyChannelId); err != nil {
		return err
	}

	return nil
}

// NewMsgSendPacket creates a new MsgSendPacket instance.
func NewMsgSendPacket(sourceChannel string, timeoutTimestamp uint64, signer string, payloads ...Payload) *MsgSendPacket {
	return &MsgSendPacket{
		SourceChannel:    sourceChannel,
		TimeoutTimestamp: timeoutTimestamp,
		Payloads:         payloads,
		Signer:           signer,
	}
}

// ValidateBasic performs basic checks on a MsgSendPacket.
func (msg *MsgSendPacket) ValidateBasic() error {
	if err := host.ChannelIdentifierValidator(msg.SourceChannel); err != nil {
		return err
	}

	if msg.TimeoutTimestamp == 0 {
		return errorsmod.Wrap(channeltypesv1.ErrInvalidTimeout, "timeout must not be 0")
	}

	if len(msg.Payloads) != 1 {
		return errorsmod.Wrapf(ErrInvalidPayload, "payloads must be of length 1, got %d instead", len(msg.Payloads))
	}

	for _, pd := range msg.Payloads {
		if err := pd.ValidateBasic(); err != nil {
			return err
		}
	}

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}

// NewMsgRecvPacket creates a new MsgRecvPacket instance.
func NewMsgRecvPacket(packet Packet, proofCommitment []byte, proofHeight clienttypes.Height, signer string) *MsgRecvPacket {
	return &MsgRecvPacket{
		Packet:          packet,
		ProofCommitment: proofCommitment,
		ProofHeight:     proofHeight,
		Signer:          signer,
	}
}

// ValidateBasic performs basic checks on a MsgRecvPacket.
func (msg *MsgRecvPacket) ValidateBasic() error {
	if len(msg.ProofCommitment) == 0 {
		return errorsmod.Wrap(commitmenttypesv1.ErrInvalidProof, "proof commitment can not be empty")
	}

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return msg.Packet.ValidateBasic()
}

// NewMsgAcknowledgement creates a new MsgAcknowledgement instance
func NewMsgAcknowledgement(packet Packet, acknowledgement Acknowledgement, proofAcked []byte, proofHeight clienttypes.Height, signer string) *MsgAcknowledgement {
	return &MsgAcknowledgement{
		Packet:          packet,
		Acknowledgement: acknowledgement,
		ProofAcked:      proofAcked,
		ProofHeight:     proofHeight,
		Signer:          signer,
	}
}

// ValidateBasic performs basic checks on a MsgAcknowledgement.
func (msg *MsgAcknowledgement) ValidateBasic() error {
	if len(msg.ProofAcked) == 0 {
		return errorsmod.Wrap(commitmenttypesv1.ErrInvalidProof, "cannot submit an empty acknowledgement proof")
	}

	// TODO: Add validation for ack object https://github.com/cosmos/ibc-go/issues/7472

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return msg.Packet.ValidateBasic()
}

// NewMsgTimeout creates a new MsgTimeout instance
func NewMsgTimeout(packet Packet, proofUnreceived []byte, proofHeight clienttypes.Height, signer string) *MsgTimeout {
	return &MsgTimeout{
		Packet:          packet,
		ProofUnreceived: proofUnreceived,
		ProofHeight:     proofHeight,
		Signer:          signer,
	}
}

// ValidateBasic performs basic checks on a MsgTimeout
func (msg *MsgTimeout) ValidateBasic() error {
	if len(msg.ProofUnreceived) == 0 {
		return errorsmod.Wrap(commitmenttypesv1.ErrInvalidProof, "proof unreceived can not be empty")
	}

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return msg.Packet.ValidateBasic()
}
