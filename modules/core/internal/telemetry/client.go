package telemetry

import (
	metrics "github.com/hashicorp/go-metrics"

	"github.com/cosmos/cosmos-sdk/telemetry"

	ibcmetrics "github.com/cosmos/ibc-go/v11/modules/core/metrics"
)

const (
	metricNameClient = "client"
	metricNameIBC    = "ibc"
)

func ReportCreateClient(clientType string) {
	telemetry.IncrCounterWithLabels(
		[]string{metricNameIBC, metricNameClient, "create"},
		1,
		[]metrics.Label{telemetry.NewLabel(ibcmetrics.LabelClientType, clientType)},
	)
}

func ReportUpdateClient(foundMisbehaviour bool, clientType, clientID string) {
	labels := []metrics.Label{
		telemetry.NewLabel(ibcmetrics.LabelClientType, clientType),
		telemetry.NewLabel(ibcmetrics.LabelClientID, clientID),
	}

	var updateType string
	if foundMisbehaviour {
		labels = append(labels, telemetry.NewLabel(ibcmetrics.LabelMsgType, "update"))
		updateType = "misbehaviour"
	} else {
		labels = append(labels, telemetry.NewLabel(ibcmetrics.LabelUpdateType, "msg"))
		updateType = "update"
	}

	telemetry.IncrCounterWithLabels([]string{metricNameIBC, metricNameClient, updateType}, 1, labels)
}

func ReportUpgradeClient(clientType, clientID string) {
	telemetry.IncrCounterWithLabels(
		[]string{metricNameIBC, metricNameClient, "upgrade"},
		1,
		[]metrics.Label{
			telemetry.NewLabel(ibcmetrics.LabelClientType, clientType),
			telemetry.NewLabel(ibcmetrics.LabelClientID, clientID),
		},
	)
}

func ReportRecoverClient(clientType, subjectClientID string) {
	telemetry.IncrCounterWithLabels(
		[]string{metricNameIBC, metricNameClient, "update"},
		1,
		[]metrics.Label{
			telemetry.NewLabel(ibcmetrics.LabelClientType, clientType),
			telemetry.NewLabel(ibcmetrics.LabelClientID, subjectClientID),
			telemetry.NewLabel(ibcmetrics.LabelUpdateType, "recovery"),
		},
	)
}
