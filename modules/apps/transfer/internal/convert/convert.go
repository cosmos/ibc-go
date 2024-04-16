package convert

import (
	"strings"

	v1types "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	v3types "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types/v3"
)

// PacketDataV1ToV3 converts a v1 (ICS20-V1) packet data to a v3 (ICS20-V2) packet data.
func PacketDataV1ToV3(packetData v1types.FungibleTokenPacketData) v3types.FungibleTokenPacketData {
	if err := packetData.ValidateBasic(); err != nil {
		panic(err)
	}

	v2Denom, trace := extractDenomAndTraceFromV1Denom(packetData.Denom)
	return v3types.FungibleTokenPacketData{
		Tokens: []*v3types.Token{
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
func extractDenomAndTraceFromV1Denom(v1Denom string) (string, []string) {
	v1DenomTrace := v1types.ParseDenomTrace(v1Denom)

	splitPath := strings.Split(v1DenomTrace.Path, "/")

	// if the path slice is empty, then the base denom is the full native denom.
	if len(splitPath) == 0 {
		return v1DenomTrace.BaseDenom, nil
	}

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
