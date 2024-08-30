package types

import (
	errorsmod "cosmossdk.io/errors"

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

// NewCounterparty creates a new Counterparty instance
func NewCounterparty(clientID string, counterpartyPacketPath commitmenttypes.MerklePath) Counterparty {
	return Counterparty{
		ClientId:               clientID,
		CounterpartyPacketPath: counterpartyPacketPath,
	}
}

// Validate validates the Counterparty
func (c Counterparty) Validate() error {
	if err := host.ClientIdentifierValidator(c.ClientId); err != nil {
		return err
	}

	if c.CounterpartyPacketPath.Empty() {
		return errorsmod.Wrap(ErrInvalidCounterparty, "prefix cannot be empty")
	}

	return nil
}
