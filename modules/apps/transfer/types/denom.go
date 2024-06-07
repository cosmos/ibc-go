package types

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	errorsmod "cosmossdk.io/errors"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
)

// NewDenom creates a new Denom instance given the base denomination and a variable number of traces.
func NewDenom(base string, traces ...Trace) Denom {
	return Denom{
		Base:  base,
		Trace: traces,
	}
}

// Validate performs a basic validation of the Denom fields.
func (d Denom) Validate() error {
	// NOTE: base denom validation cannot be performed as each chain may define
	// its own base denom validation
	if strings.TrimSpace(d.Base) == "" {
		return errorsmod.Wrap(ErrInvalidDenomForTransfer, "base denomination cannot be blank")
	}

	for _, trace := range d.Trace {
		if err := trace.Validate(); err != nil {
			return errorsmod.Wrap(err, "invalid trace")
		}
	}

	return nil
}

// Hash returns the hex bytes of the SHA256 hash of the Denom fields using the following formula:
//
// hash = sha256(trace + "/" + baseDenom)
func (d Denom) Hash() cmtbytes.HexBytes {
	hash := sha256.Sum256([]byte(d.Path()))
	return hash[:]
}

// IBCDenom a coin denomination for an ICS20 fungible token in the format
// 'ibc/{hash(trace + baseDenom)}'. If the trace is empty, it will return the base denomination.
func (d Denom) IBCDenom() string {
	if d.IsNative() {
		return d.Base
	}

	return fmt.Sprintf("%s/%s", DenomPrefix, d.Hash())
}

// Path returns the full denomination according to the ICS20 specification:
// trace + "/" + baseDenom
// If there exists no trace then the base denomination is returned.
func (d Denom) Path() string {
	if d.IsNative() {
		return d.Base
	}

	var sb strings.Builder
	for _, t := range d.Trace {
		sb.WriteString(t.String()) // nolint:revive // no error returned by WriteString
		sb.WriteByte('/')          //nolint:revive // no error returned by WriteByte
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
	// sender chain is source, if the receiver is not source
	return !d.ReceiverChainIsSource(sourcePort, sourceChannel)
}

// ReceiverChainIsSource returns true if the denomination originally came
// from the receiving chain and false otherwise.
func (d Denom) ReceiverChainIsSource(sourcePort, sourceChannel string) bool {
	// if the denom is native, then the receiver chain cannot be source
	if d.IsNative() {
		return false
	}

	// if the receiver chain originally sent the token to the sender chain, then the first element of
	// the denom's trace (latest trace) will contain the SourcePort and SourceChannel.
	return d.Trace[0].PortId == sourcePort && d.Trace[0].ChannelId == sourceChannel
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
func (d Denoms) Less(i, j int) bool {
	if d[i].Base != d[j].Base {
		return d[i].Base < d[j].Base
	}

	if len(d[i].Trace) != len(d[j].Trace) {
		return len(d[i].Trace) < len(d[j].Trace)
	}

	return d[i].Path() < d[j].Path()
}

// Swap implements sort.Interface for Denoms
func (d Denoms) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

// Sort is a helper function to sort the set of denomination in-place
func (d Denoms) Sort() Denoms {
	sort.Sort(d)
	return d
}
