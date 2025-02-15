package mock

import (
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	mockv1 "github.com/cosmos/ibc-go/v10/testing/mock"
)

const (
	ModuleName = "mockv2"
)

var MockRecvPacketResult = channeltypesv2.RecvPacketResult{
	Status:          channeltypesv2.PacketStatus_Success,
	Acknowledgement: mockv1.MockAcknowledgement.Acknowledgement(),
}

func NewMockPayload(sourcePort, destPort string) channeltypesv2.Payload {
	return channeltypesv2.Payload{
		SourcePort:      sourcePort,
		DestinationPort: destPort,
		Encoding:        transfertypes.EncodingProtobuf,
		Value:           mockv1.MockPacketData,
		Version:         mockv1.Version,
	}
}

func NewErrorMockPayload(sourcePort, destPort string) channeltypesv2.Payload {
	return channeltypesv2.Payload{
		SourcePort:      sourcePort,
		DestinationPort: destPort,
		Encoding:        transfertypes.EncodingProtobuf,
		Value:           mockv1.MockFailPacketData,
		Version:         mockv1.Version,
	}
}
