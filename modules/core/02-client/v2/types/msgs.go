package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var (
	_ sdk.Msg              = (*MsgRegisterCounterparty)(nil)
	_ sdk.HasValidateBasic = (*MsgRegisterCounterparty)(nil)
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
	return host.ClientIdentifierValidator(msg.CounterpartyClientId)
}
