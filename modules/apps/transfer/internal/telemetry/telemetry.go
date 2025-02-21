package telemetry

import (
	"fmt"

	"github.com/hashicorp/go-metrics"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	coremetrics "github.com/cosmos/ibc-go/v10/modules/core/metrics"
)

func ReportTransfer(sourcePort, sourceChannel, destinationPort, destinationChannel string, token types.Token) {
	labels := []metrics.Label{
		telemetry.NewLabel(coremetrics.LabelDestinationPort, destinationPort),
		telemetry.NewLabel(coremetrics.LabelDestinationChannel, destinationChannel),
	}

	amount, ok := sdkmath.NewIntFromString(token.Amount)
	if ok && amount.IsInt64() {
		telemetry.SetGaugeWithLabels(
			[]string{"tx", "msg", "ibc", "transfer"},
			float32(amount.Int64()),
			[]metrics.Label{telemetry.NewLabel(coremetrics.LabelDenom, token.Denom.Path())},
		)
	}

	labels = append(labels, telemetry.NewLabel(coremetrics.LabelSource, fmt.Sprintf("%t", !token.Denom.HasPrefix(sourcePort, sourceChannel))))

	telemetry.IncrCounterWithLabels(
		[]string{"ibc", types.ModuleName, "send"},
		1,
		labels,
	)
}

func ReportOnRecvPacket(sourcePort, sourceChannel, destinationPort, destinationChannel string, token types.Token) {
	labels := []metrics.Label{
		telemetry.NewLabel(coremetrics.LabelSourcePort, sourcePort),
		telemetry.NewLabel(coremetrics.LabelSourceChannel, sourceChannel),
	}

	// Modify trace as Recv does.
	if token.Denom.HasPrefix(sourcePort, sourceChannel) {
		token.Denom.Trace = token.Denom.Trace[1:]
	} else {
		trace := []types.Hop{types.NewHop(destinationPort, destinationChannel)}
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

	labels = append(labels, telemetry.NewLabel(coremetrics.LabelSource, fmt.Sprintf("%t", token.Denom.HasPrefix(sourcePort, sourceChannel))))

	telemetry.IncrCounterWithLabels(
		[]string{"ibc", types.ModuleName, "receive"},
		1,
		labels,
	)
}
