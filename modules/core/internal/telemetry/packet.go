package telemetry

import (
	metrics "github.com/hashicorp/go-metrics"

	"github.com/cosmos/cosmos-sdk/telemetry"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcmetrics "github.com/cosmos/ibc-go/v10/modules/core/metrics"
)

func ReportRecvPacket(packet types.Packet) {
	telemetry.IncrCounterWithLabels(
		[]string{"tx", "msg", "ibc", types.EventTypeRecvPacket},
		1,
		addPacketLabels(packet),
	)
}

func ReportTimeoutPacket(packet types.Packet, timeoutType string) {
	labels := append(addPacketLabels(packet), telemetry.NewLabel(ibcmetrics.LabelTimeoutType, timeoutType))
	telemetry.IncrCounterWithLabels(
		[]string{"ibc", "timeout", "packet"},
		1,
		labels,
	)
}

func ReportAcknowledgePacket(packet types.Packet) {
	telemetry.IncrCounterWithLabels(
		[]string{"tx", "msg", "ibc", types.EventTypeAcknowledgePacket},
		1,
		addPacketLabels(packet),
	)
}

func addPacketLabels(packet types.Packet) []metrics.Label {
	return []metrics.Label{
		telemetry.NewLabel(ibcmetrics.LabelSourcePort, packet.SourcePort),
		telemetry.NewLabel(ibcmetrics.LabelSourceChannel, packet.SourceChannel),
		telemetry.NewLabel(ibcmetrics.LabelDestinationPort, packet.DestinationPort),
		telemetry.NewLabel(ibcmetrics.LabelDestinationChannel, packet.DestinationChannel),
	}
}
