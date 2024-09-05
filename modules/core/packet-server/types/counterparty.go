package types

import (
	errorsmod "cosmossdk.io/errors"

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

// NewCounterparty creates a new Counterparty instance
func NewCounterparty(clientID, channelID string, merklePathPrefix commitmenttypes.MerklePath) Counterparty {
	return Counterparty{
		ClientId:            clientID,
		CounterpartyChannel: channelID,
		MerklePathPrefix:    merklePathPrefix,
	}
}

// Validate validates the Counterparty
func (c Counterparty) Validate() error {
	if err := host.ClientIdentifierValidator(c.ClientId); err != nil {
		return err
	}

	if err := host.ChannelIdentifierValidator(c.CounterpartyChannel); err != nil {
		return err
	}

	if c.MerklePathPrefix.Empty() {
		return errorsmod.Wrap(ErrInvalidCounterparty, "prefix cannot be empty")
	}

	return nil
}
