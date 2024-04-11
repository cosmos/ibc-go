package v3

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

// ValidateToken validates a token denomination and trace identifiers.
func ValidateToken(token Token) error {
	if err := sdk.ValidateDenom(token.Denom); err != nil {
		return errorsmod.Wrap(types.ErrInvalidDenomForTransfer, err.Error())
	}
	// TODO: validate the trace identifiers
	return nil
}
