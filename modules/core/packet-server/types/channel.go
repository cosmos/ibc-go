package types

import (
	errorsmod "cosmossdk.io/errors"

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

// NewChannel creates a new Channel instance
func NewChannel(clientID, counterpartyChannelID string, merklePathPrefix commitmenttypes.MerklePath) Channel {
	return Channel{
		ClientId:              clientID,
		CounterpartyChannelId: counterpartyChannelID,
		MerklePathPrefix:      merklePathPrefix,
	}
}

// Validate validates the Channel
func (c Channel) Validate() error {
	if err := host.ClientIdentifierValidator(c.ClientId); err != nil {
		return err
	}

	if err := host.ChannelIdentifierValidator(c.CounterpartyChannelId); err != nil {
		return err
	}

	if err := c.MerklePathPrefix.ValidateAsPrefix(); err != nil {
		return errorsmod.Wrap(ErrInvalidChannel, err.Error())
	}

	return nil
}
