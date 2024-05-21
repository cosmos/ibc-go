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

// Tokens is a set of Token
type Tokens []Token

// String prints out the tokens array as a string.
// If the array is empty, an empty string is returned
func (tokens Tokens) String() string {
	if len(tokens) == 0 {
		return ""
	} else if len(tokens) == 1 {
		return tokens[0].String()
	}

	var out strings.Builder
	for _, token := range tokens[:len(tokens)-1] {
		out.WriteString(token.String()) //nolint:errcheck // no error returned by WriteString
		out.WriteByte(',')              //nolint:errcheck // no error returned by WriteByte

	}
	out.WriteString(tokens[len(tokens)-1].String()) //nolint:errcheck
	return out.String()
}
