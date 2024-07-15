package metrics

// Prometheus metric labels.
const (
	// 02-client labels

	LabelClientType = "client_type"
	LabelClientID   = "client_id"
	LabelUpdateType = "update_type"
	LabelMsgType    = "msg_type"

	// Message server labels

	LabelSourcePort         = "source_port"
	LabelSourceChannel      = "source_channel"
	LabelDestinationPort    = "destination_port"
	LabelDestinationChannel = "destination_channel"
	LabelTimeoutType        = "timeout_type"
	LabelDenom              = "denom"
	LabelSource             = "source"
)
