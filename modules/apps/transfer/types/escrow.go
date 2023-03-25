package types

import (
	"fmt"
	"sort"
)

// Escrows defines a wrapper type for a slice of DenomEscrow.
type Escrows []DenomEscrow

// Validate performs a basic validation of each denomination escrow info.
func (de Escrows) Validate() error {
	seenDenoms := make(map[string]bool)
	for _, denomEscrows := range de {
		denom := denomEscrows.Denom
		if seenDenoms[denom] {
			return fmt.Errorf("duplicated denomination %s", denom)
		}

		seenDenoms[denom] = true
	}
	return nil
}

var _ sort.Interface = Escrows{}

// Len implements sort.Interface for Escrows
func (de Escrows) Len() int { return len(de) }

// Less implements sort.Interface for Escrows
func (de Escrows) Less(i, j int) bool { return de[i].Denom < de[j].Denom }

// Swap implements sort.Interface for Escrows
func (de Escrows) Swap(i, j int) { de[i], de[j] = de[j], de[i] }

// Sort is a helper function to sort the set of denomination escrows in-place
func (de Escrows) Sort() Escrows {
	sort.Sort(de)
	return de
}
