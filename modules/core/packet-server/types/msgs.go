package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var (
	_ sdk.Msg              = (*MsgProvideCounterparty)(nil)
	_ sdk.HasValidateBasic = (*MsgProvideCounterparty)(nil)
)

// NewMsgProvideCounterparty creates a new MsgProvideCounterparty instance
func NewMsgProvideCounterparty(signer, clientID, counterpartyID string, merklePathPrefix commitmenttypes.MerklePath) *MsgProvideCounterparty {
	counterparty := NewCounterparty(counterpartyID, merklePathPrefix)

	return &MsgProvideCounterparty{
		Signer:       signer,
		ClientId:     clientID,
		Counterparty: counterparty,
	}
}

// ValidateBasic performs basic checks on a MsgProvideCounterparty.
func (msg *MsgProvideCounterparty) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	if err := host.ClientIdentifierValidator(msg.ClientId); err != nil {
		return err
	}

	if err := msg.Counterparty.Validate(); err != nil {
		return err
	}

	return nil
}
