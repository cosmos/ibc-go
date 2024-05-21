package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	denominternal "github.com/cosmos/ibc-go/v8/modules/apps/transfer/internal/denom"
)

// Validate validates a token denomination and trace identifiers.
func (t Token) Validate() error {
	if err := sdk.ValidateDenom(t.Denom); err != nil {
		return errorsmod.Wrap(ErrInvalidDenomForTransfer, err.Error())
	}

	if len(t.Trace) == 0 {
		return nil
	}

	trace := strings.Join(t.Trace, "/")
	identifiers := strings.Split(trace, "/")

	return denominternal.ValidateTraceIdentifiers(identifiers)
}

// GetFullDenomPath returns the full denomination according to the ICS20 specification:
// tracePath + "/" + baseDenom
// If there exists no trace then the base denomination is returned.
func (t Token) GetFullDenomPath() string {
	trace := strings.Join(t.Trace, "/")
	if len(trace) == 0 {
		return t.Denom
	}

	return strings.Join(append(t.Trace, t.Denom), "/")
}
