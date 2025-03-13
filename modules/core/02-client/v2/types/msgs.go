package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

var (
	_ sdk.Msg = (*MsgRegisterCounterparty)(nil)
	_ sdk.Msg = (*MsgUpdateClientConfig)(nil)

	_ sdk.HasValidateBasic = (*MsgRegisterCounterparty)(nil)
	_ sdk.HasValidateBasic = (*MsgUpdateClientConfig)(nil)
)

// NewMsgRegisterCounterparty creates a new instance of MsgRegisterCounterparty.
func NewMsgRegisterCounterparty(clientID string, merklePrefix [][]byte, counterpartyClientID string, signer string) *MsgRegisterCounterparty {
	return &MsgRegisterCounterparty{
		ClientId:                 clientID,
		CounterpartyMerklePrefix: merklePrefix,
		CounterpartyClientId:     counterpartyClientID,
		Signer:                   signer,
	}
}

// ValidateBasic performs basic checks on a MsgRegisterCounterparty.
func (msg *MsgRegisterCounterparty) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	if len(msg.CounterpartyMerklePrefix) == 0 {
		return errorsmod.Wrap(ErrInvalidCounterparty, "counterparty messaging key cannot be empty")
	}
	if err := host.ClientIdentifierValidator(msg.ClientId); err != nil {
		return err
	}
	if err := host.ClientIdentifierValidator(msg.CounterpartyClientId); err != nil {
		return err
	}
	// This check must be done because the transfer v2 module assumes that the client IDs in the packet
	// are in the format {clientID}-{sequence}
	if !types.IsValidClientID(msg.ClientId) || !types.IsValidClientID(msg.CounterpartyClientId) {
		return errorsmod.Wrapf(host.ErrInvalidID, "%s and %s must be in valid format: {string}-{number}", msg.ClientId, msg.CounterpartyClientId)
	}
	return nil
}

func NewMsgUpdateClientConfig(clientID string, signer string, config Config) *MsgUpdateClientConfig {
	return &MsgUpdateClientConfig{
		ClientId: clientID,
		Signer:   signer,
		Config:   config,
	}
}

// ValidateBasic performs basic validation of the MsgUpdateClientConfig fields.
func (msg *MsgUpdateClientConfig) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	if err := host.ClientIdentifierValidator(msg.ClientId); err != nil {
		return err
	}
	if !types.IsValidClientID(msg.ClientId) {
		return errorsmod.Wrapf(host.ErrInvalidID, "client ID %s must be in valid format: {string}-{number}", msg.ClientId)
	}
	return msg.Config.Validate()
}
