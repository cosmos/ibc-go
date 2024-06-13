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

func (t Token) ConvertToCoin() (sdk.Coin, error) {
	transferAmount, ok := sdkmath.NewIntFromString(t.Amount)
	if !ok {
		return sdk.Coin{}, errorsmod.Wrapf(ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", transferAmount)
	}

	coin := sdk.NewCoin(t.Denom.IBCDenom(), transferAmount)
	return coin, nil
}
