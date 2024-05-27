package types

import (
	"crypto/sha256"
	fmt "fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

	for _, trace := range t.Trace {
		if err := trace.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Hash returns the hex bytes of the SHA256 hash of the DenomTrace fields using the following formula:
//
// hash = sha256(tracePath + "/" + baseDenom)
func (t Token) Hash() cmtbytes.HexBytes {
	hash := sha256.Sum256([]byte(t.GetFullDenomPath()))
	return hash[:]
}

// IBCDenom a coin denomination for an ICS20 fungible token in the format
// 'ibc/{hash(tracePath + baseDenom)}'. If the trace is empty, it will return the base denomination.
func (t Token) IBCDenom() string {
	if t.IsNativeDenom() {
		return t.Denom
	}

	return fmt.Sprintf("%s/%s", DenomPrefix, t.Hash())
}

// IsNativeDenom returns true if the denomination is native, thus containing no trace history.
func (t Token) IsNativeDenom() bool {
	return len(t.Trace) == 0
}

func (t Token) Coin() (sdk.Coin, error) {
	amount, ok := sdkmath.NewIntFromString(t.Amount)
	if !ok {
		return sdk.Coin{}, errorsmod.Wrapf(ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", t.Amount)
	}

	return sdk.Coin{
		Denom:  t.IBCDenom(),
		Amount: amount,
	}, nil
}

// GetFullDenomPath returns the full denomination according to the ICS20 specification:
// tracePath + "/" + baseDenom
// If there exists no trace then the base denomination is returned.
func (t Token) GetFullDenomPath() string {
	if t.IsNativeDenom() {
		return t.Denom
	}

	path := t.Denom
	for _, trace := range t.Trace {
		path = trace.String() + "/" + path
	}

	return path
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
