package telemetry

import (
	"github.com/hashicorp/go-metrics"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	coretypes "github.com/cosmos/ibc-go/v8/modules/core/types"
)

func ReportTransferTelemetry(tokens types.Tokens, labels []metrics.Label) {
	for _, token := range tokens {
		amount, ok := sdkmath.NewIntFromString(token.Amount)
		if ok && amount.IsInt64() {
			telemetry.SetGaugeWithLabels(
				[]string{"tx", "msg", "ibc", "transfer"},
				float32(amount.Int64()),
				[]metrics.Label{telemetry.NewLabel(coretypes.LabelDenom, token.Denom.Path())},
			)
		}
	}

	telemetry.IncrCounterWithLabels(
		[]string{"ibc", types.ModuleName, "send"},
		1,
		labels,
	)
}

func ReportOnRecvPacketTelemetry(transferAmount sdkmath.Int, denomPath string, labels []metrics.Label) {
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
