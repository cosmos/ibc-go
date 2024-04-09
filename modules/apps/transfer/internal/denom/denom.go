package denom

import (
	"strings"

	"strings"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// ExtractPathAndBaseFromFullDenom returns the trace path and the base denom from
// the elements that constitute the complete denom.
func ExtractPathAndBaseFromFullDenom(fullDenomItems []string) ([]string, string) {
	var (
		pathSlice      []string
		baseDenomSlice []string
	)

	length := len(fullDenomItems)
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
		if i < length-1 && length > 2 && channeltypes.IsValidChannelID(fullDenomItems[i+1]) {
			pathSlice = append(pathSlice, fullDenomItems[i], fullDenomItems[i+1])
		} else {
			baseDenomSlice = fullDenomItems[i:]
			break
		}
	}

	baseDenom := strings.Join(baseDenomSlice, "/")

	return pathSlice, baseDenom
}
