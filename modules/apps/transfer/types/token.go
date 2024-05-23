package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
)

// Validate validates a token denomination and trace identifiers.
func (t Token) Validate() error {
	if strings.TrimSpace(t.Denom) == "" {
		return errorsmod.Wrap(ErrInvalidDenomForTransfer, "denom cannot be empty")
	}

	amount, ok := sdkmath.NewIntFromString(t.Amount)
	if !ok {
		return errorsmod.Wrapf(ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", t.Amount)
	}

	if !amount.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount must be strictly positive: got %d", amount)
	}

	if len(t.Trace) == 0 {
		return nil
	}

	trace := strings.Join(t.Trace, "/")
	identifiers := strings.Split(trace, "/")

	return validateTraceIdentifiers(identifiers)
}

// GetFullDenomPath returns the full denomination according to the ICS20 specification:
// tracePath + "/" + baseDenom
// If there exists no trace then the base denomination is returned.
func (t Token) GetFullDenomPath() string {
	if len(t.Trace) == 0 {
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
		out.WriteString(token.String()) // nolint:revive // no error returned by WriteString
		out.WriteByte(',')              //nolint:revive // no error returned by WriteByte

	}
	out.WriteString(tokens[len(tokens)-1].String()) //nolint:revive
	return out.String()
}
