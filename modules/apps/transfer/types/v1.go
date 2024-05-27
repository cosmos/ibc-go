package types

import (
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

// PacketDataV1ToV2 converts a v1 packet data to a v2 packet data.
func PacketDataV1ToV2(packetData FungibleTokenPacketData) FungibleTokenPacketDataV2 {
	if err := packetData.ValidateBasic(); err != nil {
		panic(err)
	}

	v2Denom, trace := extractDenomAndTraceFromV1Denom(packetData.Denom)
	return FungibleTokenPacketDataV2{
		Tokens: []Token{
			{
				Denom:  v2Denom,
				Amount: packetData.Amount,
				Trace:  trace,
			},
		},
		Sender:   packetData.Sender,
		Receiver: packetData.Receiver,
		Memo:     packetData.Memo,
	}
}

// extractDenomAndTraceFromV1Denom extracts the base denom and remaining trace from a v1 IBC denom.
func extractDenomAndTraceFromV1Denom(v1Denom string) (string, []Trace) {
	denomSplit := strings.Split(v1Denom, "/")

	if denomSplit[0] == v1Denom {
		return v1Denom, nil
	}

	var (
		traces         []Trace
		baseDenomSlice []string
	)

	length := len(denomSplit)
	for i := 0; i < length; i += 2 {
		// The IBC specification does not guarantee the expected format of the
		// destination port or destination channel identifier. A short term solution
		// to determine base denomination is to expect the channel identifier to be the
		// one ibc-go specifies. A longer term solution is to separate the path and base
		// denomination in the ICS20 packet. If an intermediate hop prefixes the full denom
		// with a channel identifier format different from our own, the base denomination
		// will be incorrectly parsed, but the token will continue to be treated correctly
		// as an IBC denomination. The hash used to store the token internally on our chain
		// will be the same value as the base denomination being correctly parsed.
		if i < length-1 && length > 2 && channeltypes.IsValidChannelID(denomSplit[i+1]) {
			traces = append(traces, NewTrace(denomSplit[i], denomSplit[i+1]))
		} else {
			baseDenomSlice = denomSplit[i:]
			break
		}
	}

	baseDenom := strings.Join(baseDenomSlice, "/")

	return baseDenom, traces
}

// validateTraceIdentifiers validates the correctness of the trace associated with a particular base denom.
// Deprecated: only use for migration
func validateTraceIdentifiers(identifiers []string) error {
	if len(identifiers) == 0 || len(identifiers)%2 != 0 {
		return fmt.Errorf("trace info must come in pairs of port and channel identifiers '{portID}/{channelID}', got the identifiers: %s", identifiers)
	}

	// validate correctness of port and channel identifiers
	for i := 0; i < len(identifiers); i += 2 {
		if err := host.PortIdentifierValidator(identifiers[i]); err != nil {
			return errorsmod.Wrapf(err, "invalid port ID at position %d", i)
		}
		if err := host.ChannelIdentifierValidator(identifiers[i+1]); err != nil {
			return errorsmod.Wrapf(err, "invalid channel ID at position %d", i)
		}
	}
	return nil
}

// Validate performs a basic validation of the DenomTrace fields.
// Deprecated: only use for migration
func (dt DenomTrace) Validate() error {
	// empty trace is accepted when token lives on the original chain
	switch {
	case dt.Path == "" && dt.BaseDenom != "":
		return nil
	case strings.TrimSpace(dt.BaseDenom) == "":
		return fmt.Errorf("base denomination cannot be blank")
	}

	// NOTE: no base denomination validation

	identifiers := strings.Split(dt.Path, "/")
	return validateTraceIdentifiers(identifiers)
}

// ValidatePrefixedDenom checks that the denomination for an IBC fungible token packet denom is correctly prefixed.
// The function will return no error if the given string follows one of the two formats:
//
//   - Prefixed denomination: '{portIDN}/{channelIDN}/.../{portID0}/{channelID0}/baseDenom'
//   - Unprefixed denomination: 'baseDenom'
//
// 'baseDenom' may or may not contain '/'s
func ValidatePrefixedDenom(denom string) error {
	baseDenom, traces := extractDenomAndTraceFromV1Denom(denom)
	if strings.TrimSpace(baseDenom) == "" {
		return errorsmod.Wrap(ErrInvalidDenomForTransfer, "base denomination cannot be blank")
	}

	for _, trace := range traces {
		if err := trace.Validate(); err != nil {
			return err
		}
	}

	return nil
}
