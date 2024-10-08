package mock

import (
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

const (
	ModuleName = "mockv2"
)

func NewMockPacketData(sourcePort, destPort string) types.PacketData {
	return types.PacketData{
		SourcePort:      sourcePort,
		DestinationPort: destPort,
		Payload: types.Payload{
			Encoding: "json",
			Value:    mock.MockPacketData,
			Version:  mock.Version,
		},
	}
}
