package types

import (
	tmmath "github.com/tendermint/tendermint/libs/math"
	"github.com/tendermint/tendermint/light"
)

// DefaultTrustLevel is the dymint light client default trust level
var DefaultTrustLevel = NewFractionFromTm(light.DefaultTrustLevel)

// NewFractionFromTm returns a new Fraction instance from a tmmath.Fraction
func NewFractionFromTm(f tmmath.Fraction) Fraction {
	return Fraction{
		Numerator:   f.Numerator,
		Denominator: f.Denominator,
	}
}

// ToDymint converts Fraction to tmmath.Fraction
func (f Fraction) ToDymint() tmmath.Fraction {
	return tmmath.Fraction{
		Numerator:   f.Numerator,
		Denominator: f.Denominator,
	}
}
