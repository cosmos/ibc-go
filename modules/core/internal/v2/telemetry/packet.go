package telemetry

import (
	metrics "github.com/hashicorp/go-metrics"

	"github.com/cosmos/cosmos-sdk/telemetry"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibcmetrics "github.com/cosmos/ibc-go/v10/modules/core/metrics"
)

func ReportRecvPacket(packet types.Packet) {
	for _, payload := range packet.Payloads {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "ibc", types.EventTypeRecvPacket},
			1,
			[]metrics.Label{
				telemetry.NewLabel(ibcmetrics.LabelSourcePort, payload.SourcePort),
				telemetry.NewLabel(ibcmetrics.LabelSourceChannel, packet.SourceClient),
				telemetry.NewLabel(ibcmetrics.LabelDestinationPort, payload.DestinationPort),
				telemetry.NewLabel(ibcmetrics.LabelDestinationChannel, packet.DestinationClient),
			},
		)
	}
}

func ReportTimeoutPacket(packet types.Packet) {
	for _, payload := range packet.Payloads {
		telemetry.IncrCounterWithLabels(
			[]string{"ibc", "timeout", "packet"},
			1,
			[]metrics.Label{
				telemetry.NewLabel(ibcmetrics.LabelSourcePort, payload.SourcePort),
				telemetry.NewLabel(ibcmetrics.LabelSourceChannel, packet.SourceClient),
				telemetry.NewLabel(ibcmetrics.LabelDestinationPort, payload.DestinationPort),
				telemetry.NewLabel(ibcmetrics.LabelDestinationChannel, packet.DestinationClient),
				telemetry.NewLabel(ibcmetrics.LabelTimeoutType, "height"),
			},
		)
	}
}

func ReportAcknowledgePacket(packet types.Packet) {
	for _, payload := range packet.Payloads {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "ibc", types.EventTypeAcknowledgePacket},
			1,
			[]metrics.Label{
				telemetry.NewLabel(ibcmetrics.LabelSourcePort, payload.SourcePort),
				telemetry.NewLabel(ibcmetrics.LabelSourceChannel, packet.SourceClient),
				telemetry.NewLabel(ibcmetrics.LabelDestinationPort, payload.DestinationPort),
				telemetry.NewLabel(ibcmetrics.LabelDestinationChannel, packet.DestinationClient),
			},
		)
	}
}
