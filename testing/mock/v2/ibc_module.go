package mock

import (
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

func (im IBCModule) OnSendPacket(ctx context.Context, sourceID string, destinationID string, sequence uint64, data channeltypesv2.PacketData, signer sdk.AccAddress) error {
	if im.IBCApp.OnSendPacket != nil {
		return im.IBCApp.OnSendPacket(ctx, sourceID, destinationID, sequence, data, signer)
	}
	return nil
}

func (im IBCModule) OnRecvPacket(ctx context.Context, sourceID string, destinationID string, data channeltypesv2.PacketData, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
	if im.IBCApp.OnRecvPacket != nil {
		return im.IBCApp.OnRecvPacket(ctx, sourceID, destinationID, data, relayer)
	}
	return channeltypesv2.RecvPacketResult{
		Status:          channeltypesv2.PacketStatus_Success,
		Acknowledgement: mockv1.MockPacketData,
	}
}

func (im IBCModule) OnAcknowledgementPacket(ctx context.Context, sourceID string, destinationID string, data channeltypesv2.PacketData, acknowldgement []byte, relayer sdk.AccAddress) error {
	if im.IBCApp.OnAcknowledgementPacket != nil {
		return im.IBCApp.OnAcknowledgementPacket(ctx, sourceID, destinationID, data, acknowldgement, relayer)
	}
	return nil
}

func (im IBCModule) OnTimeoutPacket(ctx context.Context, sourceID string, destinationID string, data channeltypesv2.PacketData, relayer sdk.AccAddress) error {
	if im.IBCApp.OnTimeoutPacket != nil {
		return im.IBCApp.OnTimeoutPacket(ctx, sourceID, destinationID, data, relayer)
	}
	return nil
}
