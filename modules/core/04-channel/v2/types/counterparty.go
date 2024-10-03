package types

import (
	errorsmod "cosmossdk.io/errors"

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

// NewCounterparty creates a new Counterparty instance
func NewCounterparty(clientID, counterpartyChannelID string, merklePathPrefix commitmenttypes.MerklePath) Counterparty {
	return Counterparty{
		ClientId:              clientID,
		CounterpartyChannelId: counterpartyChannelID,
		MerklePathPrefix:      merklePathPrefix,
	}
}

// Validate validates the Counterparty
func (c Counterparty) Validate() error {
	if err := host.ClientIdentifierValidator(c.ClientId); err != nil {
		return err
	}

	if err := host.ChannelIdentifierValidator(c.CounterpartyChannelId); err != nil {
		return err
	}

	if err := c.MerklePathPrefix.ValidateAsPrefix(); err != nil {
		return errorsmod.Wrap(ErrInvalidCounterparty, err.Error())
	}

	return nil
}
