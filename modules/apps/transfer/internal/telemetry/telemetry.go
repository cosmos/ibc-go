package telemetry

import (
	"fmt"

	"github.com/hashicorp/go-metrics"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	coretypes "github.com/cosmos/ibc-go/v8/modules/core/types"
)

func ReportTransfer(sourcePort, sourceChannel, destinationPort, destinationChannel string, tokens types.Tokens) {
	labels := []metrics.Label{
		telemetry.NewLabel(coretypes.LabelDestinationPort, destinationPort),
		telemetry.NewLabel(coretypes.LabelDestinationChannel, destinationChannel),
	}

	for _, token := range tokens {
		amount, ok := sdkmath.NewIntFromString(token.Amount)
		if ok && amount.IsInt64() {
			telemetry.SetGaugeWithLabels(
				[]string{"tx", "msg", "ibc", "transfer"},
				float32(amount.Int64()),
				[]metrics.Label{telemetry.NewLabel(coretypes.LabelDenom, token.Denom.Path())},
			)
		}

		labels = append(labels, telemetry.NewLabel(coretypes.LabelSource, fmt.Sprintf("%t", !token.Denom.HasPrefix(sourcePort, sourceChannel))))
	}

	telemetry.IncrCounterWithLabels(
		[]string{"ibc", types.ModuleName, "send"},
		1,
		labels,
	)
}

func ReportOnRecvPacket(sourcePort, sourceChannel string, token types.Token) {
	labels := []metrics.Label{
		telemetry.NewLabel(coretypes.LabelSourcePort, sourcePort),
		telemetry.NewLabel(coretypes.LabelSourceChannel, sourceChannel),
		telemetry.NewLabel(coretypes.LabelSource, fmt.Sprintf("%t", token.Denom.HasPrefix(sourcePort, sourceChannel))),
	}
	// Transfer amount has already been parsed in caller.
	transferAmount, _ := sdkmath.NewIntFromString(token.Amount)
	denomPath := token.Denom.Path()

	if transferAmount.IsInt64() {
		telemetry.SetGaugeWithLabels(
			[]string{"ibc", types.ModuleName, "packet", "receive"},
			float32(transferAmount.Int64()),
			[]metrics.Label{telemetry.NewLabel(coretypes.LabelDenom, denomPath)},
		)
	}

	telemetry.IncrCounterWithLabels(
		[]string{"ibc", types.ModuleName, "receive"},
		1,
		labels,
	)
}
