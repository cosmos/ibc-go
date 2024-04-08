package convert

import (
	v1types "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	v3types "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types/v3"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"strconv"
	"strings"
)

// PacketDataV1ToV3 converts a v1 packet data to a v2 packet data.
func PacketDataV1ToV3(packetData v1types.FungibleTokenPacketData) v3types.FungibleTokenPacketData {
	amount, err := strconv.ParseUint(packetData.Amount, 10, 64)
	if err != nil {
		panic(err)
	}

	v2Denom, trace := extractDenomAndTraceFromV1Denom(packetData.Denom)

	if v2Denom == "" {
		panic("invalid packet data, base denom cannot be empty")
	}

	return v3types.FungibleTokenPacketData{
		Tokens: []*v3types.Token{
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

// extractDenomAndTraceFromV1Denom extracts the base denom and remaining trace from a v1 IBC denom.
func extractDenomAndTraceFromV1Denom(v1Denom string) (string, []string) {
	v1DenomTrace := v1types.ParseDenomTrace(v1Denom)

	splitPath := strings.Split(v1Denom, "/")
	pathSlice := extractPathAndBaseFromFullDenomSlice(splitPath)

	// if the path slice is empty, then the base denom is the full native denom.
	if len(pathSlice) == 0 {
		return v1DenomTrace.BaseDenom, []string(nil)
	}

	// this condition should never be reached.
	if len(pathSlice)%2 != 0 {
		panic("pathSlice length is not even")
	}

	// the path slices consists of entries of ports and channel ids separately,
	// we need to combine them to form the trace.
	var trace []string
	for i := 0; i < len(pathSlice); i += 2 {
		trace = append(trace, strings.Join(pathSlice[i:i+2], "/"))
	}

	return v1DenomTrace.BaseDenom, trace
}

// extractPathAndBaseFromFullDenomSlice extracts the path and base denom from a full denom slice.
func extractPathAndBaseFromFullDenomSlice(fullDenomItems []string) []string {
	var pathSlice []string
	length := len(fullDenomItems)
	for i := 0; i < length; i += 2 {
		if i < length-1 && length > 2 && channeltypes.IsValidChannelID(fullDenomItems[i+1]) {
			pathSlice = append(pathSlice, fullDenomItems[i], fullDenomItems[i+1])
		} else {
			return pathSlice
		}
	}
	return pathSlice
}
