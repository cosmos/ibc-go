package telemetry

import (
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// ReportRecvPacket TODO: https://github.com/cosmos/ibc-go/issues/7437
func ReportRecvPacket(packet channeltypesv2.Packet) {}

// ReportTimeoutPacket TODO: https://github.com/cosmos/ibc-go/issues/7437
func ReportTimeoutPacket(packet channeltypesv2.Packet, timeoutType string) {}

// ReportAcknowledgePacket TODO: https://github.com/cosmos/ibc-go/issues/7437
func ReportAcknowledgePacket(packet channeltypesv2.Packet) {}
