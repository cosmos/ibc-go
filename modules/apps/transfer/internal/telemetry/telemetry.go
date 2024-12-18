package telemetry

import (
	"fmt"

	"github.com/hashicorp/go-metrics"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	coremetrics "github.com/cosmos/ibc-go/v9/modules/core/metrics"
)

func ReportTransfer(sourcePort, sourceChannel, destinationPort, destinationChannel string, tokens types.Tokens) {
	labels := []metrics.Label{
		telemetry.NewLabel(coremetrics.LabelDestinationPort, destinationPort),
		telemetry.NewLabel(coremetrics.LabelDestinationChannel, destinationChannel),
	}

	for _, token := range tokens {
		amount, ok := sdkmath.NewIntFromString(token.Amount)
		if ok && amount.IsInt64() {
			telemetry.SetGaugeWithLabels(
				[]string{"tx", "msg", "ibc", "transfer"},
				float32(amount.Int64()),
				[]metrics.Label{telemetry.NewLabel(coremetrics.LabelDenom, token.Denom.Path())},
			)
		}

		labels = append(labels, telemetry.NewLabel(coremetrics.LabelSource, fmt.Sprintf("%t", !token.Denom.HasPrefix(sourcePort, sourceChannel))))
	}

	telemetry.IncrCounterWithLabels(
		[]string{"ibc", types.ModuleName, "send"},
		1,
		labels,
	)
}

func ReportOnRecvPacket(packet channeltypes.Packet, tokens types.Tokens) {
	labels := []metrics.Label{
		telemetry.NewLabel(coremetrics.LabelSourcePort, packet.SourcePort),
		telemetry.NewLabel(coremetrics.LabelSourceChannel, packet.SourceChannel),
	}

	for _, token := range tokens {
		// Modify trace as Recv does.
		if token.Denom.HasPrefix(packet.SourcePort, packet.SourceChannel) {
			token.Denom.Trace = token.Denom.Trace[1:]
		} else {
			trace := []types.Hop{types.NewHop(packet.DestinationPort, packet.DestinationChannel)}
			token.Denom.Trace = append(trace, token.Denom.Trace...)
		}

		// Transfer amount has already been parsed in caller.
		transferAmount, ok := sdkmath.NewIntFromString(token.Amount)
		if ok && transferAmount.IsInt64() {
			telemetry.SetGaugeWithLabels(
				[]string{"ibc", types.ModuleName, "packet", "receive"},
				float32(transferAmount.Int64()),
				[]metrics.Label{telemetry.NewLabel(coremetrics.LabelDenom, token.Denom.Path())},
			)
		}

		labels = append(labels, telemetry.NewLabel(coremetrics.LabelSource, fmt.Sprintf("%t", token.Denom.HasPrefix(packet.SourcePort, packet.SourceChannel))))
	}

	telemetry.IncrCounterWithLabels(
		[]string{"ibc", types.ModuleName, "receive"},
		1,
		labels,
	)
}
