package types

import (
	"crypto/sha256"
	fmt "fmt"
	"sort"

	errorsmod "cosmossdk.io/errors"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
)

// Hash returns the hex bytes of the SHA256 hash of the Denom fields using the following formula:
//
// hash = sha256(tracePath + "/" + baseDenom)
func (d Denom) Hash() cmtbytes.HexBytes {
	hash := sha256.Sum256([]byte(d.GetFullDenomPath()))
	return hash[:]
}

// IBCDenom a coin denomination for an ICS20 fungible token in the format
// 'ibc/{hash(tracePath + baseDenom)}'. If the trace is empty, it will return the base denomination.
func (d DenomTrace) IBCDenom() string {
	if d.IsNative() {
		return d.BaseDenom
	}

	return fmt.Sprintf("%s/%s", DenomPrefix, d.Hash())
}

// GetFullPath returns the full denomination according to the ICS20 specification:
// tracePath + "/" + baseDenom
// If there exists no trace then the base denomination is returned.
func (d Denom) GetFullPath() string {
	if d.IsNative() {
		return d.Base
	}

	path := d.Trace[0]
	for i := 1; i < len(d.Trace)-1; i++ {
		path = fmt.Sprintf("%s/%s", path, d.Trace[i])
	}

	return fmt.Sprintf("%s/%s", path, d.Base)
}

// IsNative returns true if the denomination is native, thus containing no trace history.
func (d Denom) IsNative() bool {
	return len(d.Trace) == 0
}

// Denoms defines a wrapper type for a slice of Denom.
type Denoms []Denom

// Validate performs a basic validation of each denomination trace info.
func (d Denoms) Validate() error {
	seenDenoms := make(map[string]bool)
	for i, denom := range d {
		hash := denom.Hash().String()
		if seenDenom[hash] {
			return fmt.Errorf("duplicated denomination trace with hash %s", denom.Hash())
		}

		if err := denom.Validate(); err != nil {
			return errorsmod.Wrapf(err, "failed denom trace %d validation", i)
		}
		seenDenom[hash] = true
	}
	return nil
}

var _ sort.Interface = (*Denoms)(nil)

// Len implements sort.Interface for Denoms
func (d Denoms) Len() int { return len(t) }

// Less implements sort.Interface for Denoms
func (d Denoms) Less(i, j int) bool { return d[i].GetFullPath() < d[j].GetFullPath() }

// Swap implements sort.Interface for Denoms
func (d Denoms) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

// Sort is a helper function to sort the set of denomination traces in-place
func (d Denoms) Sort() Denoms {
	sort.Sort(t)
	return d
}
