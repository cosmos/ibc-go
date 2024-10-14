package mock

import (
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

const (
	ModuleName = "mockv2"
)

var (
	MockAcknowledgement = types.Acknowledgement{
		AcknowledgementResults: []types.AcknowledgementResult{{
			AppName: ModuleName,
			RecvPacketResult: types.RecvPacketResult{
				Status:          types.PacketStatus_Success,
				Acknowledgement: mock.MockAcknowledgement.Acknowledgement(),
			},
		}},
	}
	MockFailAcknowledgement = types.Acknowledgement{
		AcknowledgementResults: []types.AcknowledgementResult{{
			AppName: ModuleName,
			RecvPacketResult: types.RecvPacketResult{
				Status:          types.PacketStatus_Failure,
				Acknowledgement: mock.MockFailAcknowledgement.Acknowledgement(),
			},
		}},
	}
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
