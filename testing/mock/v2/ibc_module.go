package mock

import (
	"bytes"
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/api"
	mockv1 "github.com/cosmos/ibc-go/v9/testing/mock"
)

var _ api.IBCModule = (*IBCModule)(nil)

const (
	// ModuleNameA is a name that can be used for the first mock application.
	ModuleNameA = ModuleName + "A"
	// ModuleNameB is a name that can be used for the second mock application.
	ModuleNameB = ModuleName + "B"
	// PortIDA is a port ID that can be used for the first mock application.
	PortIDA = ModuleNameA
	// PortIDB is a port ID that can be used for the second mock application.
	PortIDB = ModuleNameB
)

// IBCModule is a mock implementation of the IBCModule interface.
// which delegates calls to the underlying IBCApp.
type IBCModule struct {
	IBCApp *IBCApp
}

// NewIBCModule creates a new IBCModule with an underlying mock IBC application.
func NewIBCModule() IBCModule {
	return IBCModule{
		IBCApp: &IBCApp{},
	}
}

func (im IBCModule) OnSendPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, data channeltypesv2.Payload, signer sdk.AccAddress) error {
	if im.IBCApp.OnSendPacket != nil {
		return im.IBCApp.OnSendPacket(ctx, sourceChannel, destinationChannel, sequence, data, signer)
	}
	return nil
}

func (im IBCModule) OnRecvPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
	if im.IBCApp.OnRecvPacket != nil {
		return im.IBCApp.OnRecvPacket(ctx, sourceChannel, destinationChannel, sequence, payload, relayer)
	}
	if bytes.Equal(payload.Value, mockv1.MockPacketData) {
		return MockRecvPacketResult
	}
	return channeltypesv2.RecvPacketResult{Status: channeltypesv2.PacketStatus_Failure}
}

func (im IBCModule) OnAcknowledgementPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	if im.IBCApp.OnAcknowledgementPacket != nil {
		return im.IBCApp.OnAcknowledgementPacket(ctx, sourceChannel, destinationChannel, sequence, payload, acknowledgement, relayer)
	}
	return nil
}

func (im IBCModule) OnTimeoutPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	if im.IBCApp.OnTimeoutPacket != nil {
		return im.IBCApp.OnTimeoutPacket(ctx, sourceChannel, destinationChannel, sequence, payload, relayer)
	}
	return nil
}

func (IBCModule) UnmarshalPacketData(payload channeltypesv2.Payload) (interface{}, error) {
	if bytes.Equal(payload.Value, mockv1.MockPacketData) {
		return mockv1.MockPacketData, nil
	}
	return nil, mockv1.MockApplicationCallbackError
}
