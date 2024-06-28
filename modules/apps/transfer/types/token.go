package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Tokens is a slice of Tokens
type Tokens []Token

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

// ToCoin converts a Token to an sdk.Coin.
//
// The function parses the Amount field of the Token into an sdkmath.Int and returns a new sdk.Coin with
// the IBCDenom of the Token's Denom field and the parsed Amount.
// If the Amount cannot be parsed, an error is returned with a wrapped error message.
func (t Token) ToCoin() (sdk.Coin, error) {
	transferAmount, ok := sdkmath.NewIntFromString(t.Amount)
	if !ok {
		return sdk.Coin{}, errorsmod.Wrapf(ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", transferAmount)
	}

	coin := sdk.NewCoin(t.Denom.IBCDenom(), transferAmount)
	return coin, nil
}
