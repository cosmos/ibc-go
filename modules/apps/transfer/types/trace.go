package types

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmttypes "github.com/cometbft/cometbft/types"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

func NewTrace(portID, channelID string) Trace {
	return Trace{
		PortId:    portID,
		ChannelId: channelID,
	}
}

// Validate performs basic validation of the trace
func (t Trace) Validate() error {
	if err := host.PortIdentifierValidator(t.PortId); err != nil {
		return errorsmod.Wrapf(err, "invalid portID")
	}
	if err := host.ChannelIdentifierValidator(t.ChannelId); err != nil {
		return errorsmod.Wrapf(err, "invalid channelID")
	}

	return nil
}

func (t Trace) String() string {
	return t.PortId + "/" + t.ChannelId
}

// Traces defines a wrapper type for a slice of DenomTrace.
type Traces []Token

// Validate performs a basic validation of each denomination trace info.
func (t Traces) Validate() error {
	seenTraces := make(map[string]bool)
	for i, trace := range t {
		hash := trace.Hash().String()
		if seenTraces[hash] {
			return fmt.Errorf("duplicated denomination trace with hash %s", trace.Hash())
		}

		if err := trace.Validate(); err != nil {
			return errorsmod.Wrapf(err, "failed denom trace %d validation", i)
		}
		seenTraces[hash] = true
	}
	return nil
}

var _ sort.Interface = (*Traces)(nil)

// Len implements sort.Interface for Traces
func (t Traces) Len() int { return len(t) }

// Less implements sort.Interface for Traces
func (t Traces) Less(i, j int) bool { return t[i].GetFullDenomPath() < t[j].GetFullDenomPath() }

// Swap implements sort.Interface for Traces
func (t Traces) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

// Sort is a helper function to sort the set of denomination traces in-place
func (t Traces) Sort() Traces {
	sort.Sort(t)
	return t
}

// validateIBCDenom validates that the given denomination is either:
//
//   - A valid base denomination (eg: 'uatom' or 'gamm/pool/1' as in https://github.com/cosmos/ibc-go/issues/894)
//   - A valid fungible token representation (i.e 'ibc/{hash}') per ADR 001 https://github.com/cosmos/ibc-go/blob/main/docs/architecture/adr-001-coin-source-tracing.md
func validateIBCDenom(denom string) error {
	if err := sdk.ValidateDenom(denom); err != nil {
		return err
	}

	denomSplit := strings.SplitN(denom, "/", 2)

	switch {
	case denom == DenomPrefix:
		return errorsmod.Wrapf(ErrInvalidDenomForTransfer, "denomination should be prefixed with the format 'ibc/{hash(trace + \"/\" + %s)}'", denom)

	case len(denomSplit) == 2 && denomSplit[0] == DenomPrefix:
		if strings.TrimSpace(denomSplit[1]) == "" {
			return errorsmod.Wrapf(ErrInvalidDenomForTransfer, "denomination should be prefixed with the format 'ibc/{hash(trace + \"/\" + %s)}'", denom)
		}

		if _, err := ParseHexHash(denomSplit[1]); err != nil {
			return errorsmod.Wrapf(err, "invalid denom trace hash %s", denomSplit[1])
		}
	}

	return nil
}

// ParseHexHash parses a hex hash in string format to bytes and validates its correctness.
func ParseHexHash(hexHash string) (cmtbytes.HexBytes, error) {
	hash, err := hex.DecodeString(hexHash)
	if err != nil {
		return nil, err
	}

	if err := cmttypes.ValidateHash(hash); err != nil {
		return nil, err
	}

	return hash, nil
}
