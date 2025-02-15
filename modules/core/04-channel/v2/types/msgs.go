package types

import (
	"time"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypesv1 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

const MaxTimeoutDelta time.Duration = 24 * time.Hour

var (
	_ sdk.Msg              = (*MsgSendPacket)(nil)
	_ sdk.HasValidateBasic = (*MsgSendPacket)(nil)

	_ sdk.Msg              = (*MsgRecvPacket)(nil)
	_ sdk.HasValidateBasic = (*MsgRecvPacket)(nil)

	_ sdk.Msg              = (*MsgTimeout)(nil)
	_ sdk.HasValidateBasic = (*MsgTimeout)(nil)

	_ sdk.Msg              = (*MsgAcknowledgement)(nil)
	_ sdk.HasValidateBasic = (*MsgAcknowledgement)(nil)
)

// NewMsgSendPacket creates a new MsgSendPacket instance.
func NewMsgSendPacket(sourceClient string, timeoutTimestamp uint64, signer string, payloads ...Payload) *MsgSendPacket {
	return &MsgSendPacket{
		SourceClient:     sourceClient,
		TimeoutTimestamp: timeoutTimestamp,
		Payloads:         payloads,
		Signer:           signer,
	}
}

// ValidateBasic performs basic checks on a MsgSendPacket.
func (msg *MsgSendPacket) ValidateBasic() error {
	if err := host.ClientIdentifierValidator(msg.SourceClient); err != nil {
		return err
	}

	if msg.TimeoutTimestamp == 0 {
		return errorsmod.Wrap(ErrInvalidTimeout, "timeout must not be 0")
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

	if err := msg.Acknowledgement.Validate(); err != nil {
		return err
	}

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
