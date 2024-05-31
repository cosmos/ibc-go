package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
)

// Validate validates a token denomination and amount.
func (t Token) Validate() error {
	if err := t.Denom.Validate(); err != nil {
		return errorsmod.Wrap(err, "invalid token denom")
	}

	amount, ok := sdkmath.NewIntFromString(t.Amount)
	if !ok {
		return errorsmod.Wrapf(ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", t.Amount)
	}

	if !amount.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount must be strictly positive: got %d", amount)
	}

	return nil
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
