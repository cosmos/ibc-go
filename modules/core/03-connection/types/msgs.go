package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	_ sdk.Msg = (*MsgConnectionOpenInit)(nil)
	_ sdk.Msg = (*MsgConnectionOpenConfirm)(nil)
	_ sdk.Msg = (*MsgConnectionOpenAck)(nil)
	_ sdk.Msg = (*MsgConnectionOpenTry)(nil)
	_ sdk.Msg = (*MsgUpdateParams)(nil)

	_ sdk.HasValidateBasic = (*MsgConnectionOpenInit)(nil)
	_ sdk.HasValidateBasic = (*MsgConnectionOpenConfirm)(nil)
	_ sdk.HasValidateBasic = (*MsgConnectionOpenAck)(nil)
	_ sdk.HasValidateBasic = (*MsgConnectionOpenTry)(nil)
	_ sdk.HasValidateBasic = (*MsgUpdateParams)(nil)
)

// NewMsgConnectionOpenInit creates a new MsgConnectionOpenInit instance. It sets the
// counterparty connection identifier to be empty.
func NewMsgConnectionOpenInit(
	clientID, counterpartyClientID string,
	counterpartyPrefix commitmenttypes.MerklePrefix,
	version *Version, delayPeriod uint64, signer string,
) *MsgConnectionOpenInit {
	// counterparty must have the same delay period
	counterparty := NewCounterparty(counterpartyClientID, "", counterpartyPrefix)
	return &MsgConnectionOpenInit{
		ClientId:     clientID,
		Counterparty: counterparty,
		Version:      version,
		DelayPeriod:  delayPeriod,
		Signer:       signer,
	}
}

// ValidateBasic implements sdk.Msg.
func (msg MsgConnectionOpenInit) ValidateBasic() error {
	if msg.ClientId == exported.LocalhostClientID {
		return errorsmod.Wrap(clienttypes.ErrInvalidClientType, "localhost connection handshakes are disallowed")
	}

	if err := host.ClientIdentifierValidator(msg.ClientId); err != nil {
		return errorsmod.Wrap(err, "invalid client ID")
	}
	if msg.Counterparty.ConnectionId != "" {
		return errorsmod.Wrap(ErrInvalidCounterparty, "counterparty connection identifier must be empty")
	}

	// NOTE: Version can be nil on MsgConnectionOpenInit
	if msg.Version != nil {
		if err := ValidateVersion(msg.Version); err != nil {
			return errorsmod.Wrap(err, "basic validation of the provided version failed")
		}
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return msg.Counterparty.ValidateBasic()
}

// NewMsgConnectionOpenTry creates a new MsgConnectionOpenTry instance
func NewMsgConnectionOpenTry(
	clientID, counterpartyConnectionID, counterpartyClientID string,
	counterpartyPrefix commitmenttypes.MerklePrefix,
	counterpartyVersions []*Version, delayPeriod uint64,
	initProof []byte, proofHeight clienttypes.Height, signer string,
) *MsgConnectionOpenTry {
	counterparty := NewCounterparty(counterpartyClientID, counterpartyConnectionID, counterpartyPrefix)
	return &MsgConnectionOpenTry{
		ClientId:             clientID,
		Counterparty:         counterparty,
		CounterpartyVersions: counterpartyVersions,
		DelayPeriod:          delayPeriod,
		ProofInit:            initProof,
		ProofHeight:          proofHeight,
		Signer:               signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgConnectionOpenTry) ValidateBasic() error {
	if msg.ClientId == exported.LocalhostClientID {
		return errorsmod.Wrap(clienttypes.ErrInvalidClientType, "localhost connection handshakes are disallowed")
	}

	if err := host.ClientIdentifierValidator(msg.ClientId); err != nil {
		return errorsmod.Wrap(err, "invalid client ID")
	}
	// counterparty validate basic allows empty counterparty connection identifiers
	if err := host.ConnectionIdentifierValidator(msg.Counterparty.ConnectionId); err != nil {
		return errorsmod.Wrap(err, "invalid counterparty connection ID")
	}
	if len(msg.CounterpartyVersions) == 0 {
		return errorsmod.Wrap(ibcerrors.ErrInvalidVersion, "empty counterparty versions")
	}
	if len(msg.CounterpartyVersions) > MaxCounterpartyVersionsLength {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidVersion, "counterparty versions must not exceed %d items", MaxCounterpartyVersionsLength)
	}
	for i, version := range msg.CounterpartyVersions {
		if err := ValidateVersion(version); err != nil {
			return errorsmod.Wrapf(err, "basic validation failed on version with index %d", i)
		}
	}
	if len(msg.ProofInit) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty proof init")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return msg.Counterparty.ValidateBasic()
}

// NewMsgConnectionOpenAck creates a new MsgConnectionOpenAck instance
func NewMsgConnectionOpenAck(
	connectionID, counterpartyConnectionID string, tryProof []byte,
	proofHeight clienttypes.Height, version *Version, signer string,
) *MsgConnectionOpenAck {
	return &MsgConnectionOpenAck{
		ConnectionId:             connectionID,
		CounterpartyConnectionId: counterpartyConnectionID,
		ProofTry:                 tryProof,
		ProofHeight:              proofHeight,
		Version:                  version,
		Signer:                   signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgConnectionOpenAck) ValidateBasic() error {
	if !IsValidConnectionID(msg.ConnectionId) {
		return ErrInvalidConnectionIdentifier
	}
	if err := host.ConnectionIdentifierValidator(msg.CounterpartyConnectionId); err != nil {
		return errorsmod.Wrap(err, "invalid counterparty connection ID")
	}
	if err := ValidateVersion(msg.Version); err != nil {
		return err
	}
	if len(msg.ProofTry) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty proof try")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return nil
}

// NewMsgConnectionOpenConfirm creates a new MsgConnectionOpenConfirm instance
func NewMsgConnectionOpenConfirm(
	connectionID string, ackProof []byte, proofHeight clienttypes.Height,
	signer string,
) *MsgConnectionOpenConfirm {
	return &MsgConnectionOpenConfirm{
		ConnectionId: connectionID,
		ProofAck:     ackProof,
		ProofHeight:  proofHeight,
		Signer:       signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgConnectionOpenConfirm) ValidateBasic() error {
	if !IsValidConnectionID(msg.ConnectionId) {
		return ErrInvalidConnectionIdentifier
	}
	if len(msg.ProofAck) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty proof ack")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return nil
}

// NewMsgUpdateParams creates a new MsgUpdateParams instance
func NewMsgUpdateParams(signer string, params Params) *MsgUpdateParams {
	return &MsgUpdateParams{
		Signer: signer,
		Params: params,
	}
}

// ValidateBasic performs basic checks on a MsgUpdateParams.
func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return msg.Params.Validate()
}
