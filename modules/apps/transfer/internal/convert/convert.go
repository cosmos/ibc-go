package convert

import (
	"strings"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

// PacketDataV1ToV2 converts a v1 packet data to a v2 packet data.
func PacketDataV1ToV2(packetData types.FungibleTokenPacketData) types.FungibleTokenPacketDataV2 {
	if err := packetData.ValidateBasic(); err != nil {
		panic(err)
	}

	v2Denom, trace := ExtractDenomAndTraceFromV1Denom(packetData.Denom)
	return types.FungibleTokenPacketDataV2{
		Tokens: []*types.Token{
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
func ExtractDenomAndTraceFromV1Denom(v1Denom string) (string, []string) {
	v1DenomTrace := types.ParseDenomTrace(v1Denom)

	// if the path string is empty, then the base denom is the full native denom.
	if v1DenomTrace.Path == "" {
		return v1DenomTrace.BaseDenom, nil
	}

	splitPath := strings.Split(v1DenomTrace.Path, "/")

	// this condition should never be reached.
	if len(splitPath)%2 != 0 {
		panic("pathSlice length is not even")
	}

	// the path slices consists of entries of ports and channel ids separately,
	// we need to combine them to form the trace.
	var trace []string
	for i := 0; i < len(splitPath); i += 2 {
		trace = append(trace, strings.Join(splitPath[i:i+2], "/"))
	}

	return v1DenomTrace.BaseDenom, trace
}
