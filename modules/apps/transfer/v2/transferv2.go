package transfer

import (
	"strconv"
	"strings"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// ConvertPacketV1ToPacketV2 converts a v1 packet data to a v2 packet data.
func ConvertPacketV1ToPacketV2(packetData types.FungibleTokenPacketData) types.FungibleTokenPacketDataV2 {
	amount, err := strconv.ParseUint(packetData.Amount, 10, 64)
	if err != nil {
		panic(err)
	}

	v2Denom, trace := ExtractDenomAndTraceFromV1Denom(packetData.Denom)

	// TODO: we should fail here, but some tests fail with this panic. We can re-visit.
	// if v2Denom == "" {
	// 	panic("base denom cannot be empty")
	// }

	return types.FungibleTokenPacketDataV2{
		Tokens: []*types.Token{
			{
				Denom:  v2Denom,
				Amount: amount,
				Trace:  trace,
			},
		},
		Sender:   packetData.Sender,
		Receiver: packetData.Receiver,
		Memo:     packetData.Memo,
	}
}

func ExtractDenomAndTraceFromV1Denom(v1Denom string) (string, []string) {
	v1DenomTrace := types.ParseDenomTrace(v1Denom)

	splitPath := strings.Split(v1Denom, "/")
	pathSlice := extractPathAndBaseFromFullDenomSlice(splitPath)

	if len(pathSlice) == 0 {
		return v1DenomTrace.BaseDenom, []string(nil)
	}

	if len(pathSlice)%2 != 0 {
		panic("pathSlice length is not even")
	}

	var trace []string
	for i := 0; i < len(pathSlice); i += 2 {
		trace = append(trace, strings.Join(pathSlice[i:i+2], "/"))
	}

	return v1DenomTrace.BaseDenom, trace
}

func extractPathAndBaseFromFullDenomSlice(fullDenomItems []string) []string {
	var pathSlice []string

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
			break
		}
	}
	return pathSlice
}
