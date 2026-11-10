// SPDX-License-Identifier: Apache-2.0

package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcerrors "github.com/cosmos/ibc-go/v11/modules/core/errors"
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
		if err := account.AccountId.Validate(); err != nil {
			return err
		}
	}

	return nil
}
