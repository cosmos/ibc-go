package telemetry

import (
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

func ReportRecvPacket(packet channeltypesv2.Packet) {}

func ReportTimeoutPacket(packet channeltypesv2.Packet, timeoutType string) {}

func ReportAcknowledgePacket(packet channeltypesv2.Packet) {}
