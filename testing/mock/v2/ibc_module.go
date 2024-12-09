package mock

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/api"
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

func (im IBCModule) OnSendPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, data channeltypesv2.Payload, signer sdk.AccAddress) error {
	if im.IBCApp.OnSendPacket != nil {
		return im.IBCApp.OnSendPacket(ctx, sourceChannel, destinationChannel, sequence, data, signer)
	}
	return nil
}

func (im IBCModule) OnRecvPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, data channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
	if im.IBCApp.OnRecvPacket != nil {
		return im.IBCApp.OnRecvPacket(ctx, sourceChannel, destinationChannel, sequence, data, relayer)
	}
	return MockRecvPacketResult
}

func (im IBCModule) OnAcknowledgementPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, recvSuccess bool, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	if im.IBCApp.OnAcknowledgementPacket != nil {
		return im.IBCApp.OnAcknowledgementPacket(ctx, sourceChannel, destinationChannel, sequence, payload, recvSuccess, acknowledgement, relayer)
	}
	return nil
}

func (im IBCModule) OnTimeoutPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	if im.IBCApp.OnTimeoutPacket != nil {
		return im.IBCApp.OnTimeoutPacket(ctx, sourceChannel, destinationChannel, sequence, payload, relayer)
	}
	return nil
}
