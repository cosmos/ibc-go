package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

// DefaultGenesisState returns the default GenesisState.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Ics27Accounts: []RegisteredICS27Account{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	for _, account := range gs.Ics27Accounts {
		if _, err := sdk.AccAddressFromBech32(account.AccountAddress); err != nil {
			return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}
		if _, err := sdk.AccAddressFromBech32(account.AccountId.Sender); err != nil {
			return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}
		if err := host.ClientIdentifierValidator(account.AccountId.ClientId); err != nil {
			return errorsmod.Wrapf(err, "invalid source client ID %s", account.AccountId.ClientId)
		}
		if len(account.AccountId.Salt) > MaximumSaltLength {
			return errorsmod.Wrapf(ErrInvalidSalt, "salt must not exceed %d bytes", MaximumSaltLength)
		}
	}

	return nil
}
