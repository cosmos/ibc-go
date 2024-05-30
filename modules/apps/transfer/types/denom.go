package types

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	errorsmod "cosmossdk.io/errors"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
)

// Validate performs a basic validation of the Denom fields.
func (d Denom) Validate() error {
	// NOTE: base denom validation cannot be performed as each chain may define
	// its own base denom validation
	if strings.TrimSpace(d.Base) == "" {
		return fmt.Errorf("base denomination cannot be blank")
	}

	if len(d.Trace) != 0 {
		trace := strings.Join(d.Trace, "/")
		identifiers := strings.Split(trace, "/")

		if err := validateTraceIdentifiers(identifiers); err != nil {
			return err
		}
	}

	return nil
}

// Hash returns the hex bytes of the SHA256 hash of the Denom fields using the following formula:
//
// hash = sha256(tracePath + "/" + baseDenom)
func (d Denom) Hash() cmtbytes.HexBytes {
	hash := sha256.Sum256([]byte(d.FullPath()))
	return hash[:]
}

// IBCDenom a coin denomination for an ICS20 fungible token in the format
// 'ibc/{hash(tracePath + baseDenom)}'. If the trace is empty, it will return the base denomination.
func (d Denom) IBCDenom() string {
	if d.IsNative() {
		return d.Base
	}

	return fmt.Sprintf("%s/%s", DenomPrefix, d.Hash())
}

// FullPath returns the full denomination according to the ICS20 specification:
// tracePath + "/" + baseDenom
// If there exists no trace then the base denomination is returned.
func (d Denom) FullPath() string {
	if d.IsNative() {
		return d.Base
	}

	var sb strings.Builder
	for _, t := range d.Trace {
		sb.WriteString(t) // nolint:revive // no error returned by WriteString
		sb.WriteByte('/') //nolint:revive // no error returned by WriteByte
	}
	sb.WriteString(d.Base) //nolint:revive
	return sb.String()
}

// IsNative returns true if the denomination is native, thus containing no trace history.
func (d Denom) IsNative() bool {
	return len(d.Trace) == 0
}

// SenderChainIsSource returns false if the denomination originally came
// from the receiving chain and true otherwise.
func (d Denom) SenderChainIsSource(sourcePort, sourceChannel string) bool {
	// This is the prefix that would have been prefixed to the denomination
	// on sender chain IF and only if the token originally came from the
	// receiving chain.

	return !d.ReceiverChainIsSource(sourcePort, sourceChannel)
}

// ReceiverChainIsSource returns true if the denomination originally came
// from the receiving chain and false otherwise.
func (d Denom) ReceiverChainIsSource(sourcePort, sourceChannel string) bool {
	// The first element in the Denom's trace should contain the SourcePort and SourceChannel.
	// If the receiver chain originally sent the token to the sender chain, the first element of
	// the denom's trace will contain the sender's SourcePort and SourceChannel.
	if d.IsNative() {
		return false
	}

	return d.Trace[0] == sourcePort+"/"+sourceChannel
}

// Denoms defines a wrapper type for a slice of Denom.
type Denoms []Denom

// Validate performs a basic validation of each denomination trace info.
func (d Denoms) Validate() error {
	seenDenoms := make(map[string]bool)
	for i, denom := range d {
		hash := denom.Hash().String()
		if seenDenoms[hash] {
			return fmt.Errorf("duplicated denomination with hash %s", denom.Hash())
		}

		if err := denom.Validate(); err != nil {
			return errorsmod.Wrapf(err, "failed denom %d validation", i)
		}
		seenDenoms[hash] = true
	}
	return nil
}

var _ sort.Interface = (*Denoms)(nil)

// Len implements sort.Interface for Denoms
func (d Denoms) Len() int { return len(d) }

// Less implements sort.Interface for Denoms
func (d Denoms) Less(i, j int) bool { return d[i].FullPath() < d[j].FullPath() }

// Swap implements sort.Interface for Denoms
func (d Denoms) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

// Sort is a helper function to sort the set of denomination in-place
func (d Denoms) Sort() Denoms {
	sort.Sort(d)
	return d
}
