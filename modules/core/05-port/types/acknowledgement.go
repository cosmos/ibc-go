package types

import "github.com/cosmos/ibc-go/v8/modules/core/exported"

func (rack RoutedPacketAcknowledgement) Acknowledgement() []byte {
	return SubModuleCdc.MustMarshalJSON(&rack)
}

func (rack RoutedPacketAcknowledgement) Success() bool {
	for _, ack := range rack.PacketAck {
		var intAck exported.Acknowledgement
		SubModuleCdc.UnmarshalInterface(ack, &intAck)
		if !intAck.Success() {
			return false
		}
	}
	return true
}
