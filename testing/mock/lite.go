package mock

import (
	"bytes"
	"context"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	litetypes "github.com/cosmos/ibc-go/v8/modules/core/lite/types"
)

var _ litetypes.PacketModule = (*LitePacketModule)(nil)

type LitePacketModule struct{}

func (LitePacketModule) OnSendPacket(
	ctx context.Context,
	sourcePort string,
	sourceChannel string,
	sequence uint64,
	version string,
	data []byte,
	sender string,
) error {
	if bytes.Equal(data, MockAsyncPacketData) || bytes.Equal(data, MockPacketData) || bytes.Equal(data, MockFailPacketData) {
		return nil
	}
	return MockApplicationCallbackError
}

// OnRecvPacket must return an acknowledgement that implements the Acknowledgement interface.
// In the case of an asynchronous acknowledgement, nil should be returned.
// If the acknowledgement returned is successful, the state changes on callback are written,
// otherwise the application state changes are discarded. In either case the packet is received
// and the acknowledgement is written (in synchronous cases).
func (LitePacketModule) OnRecvPacket(
	ctx context.Context,
	version string,
	packet exported.PacketI,
	relayer string,
) exported.Acknowledgement {
	if bytes.Equal(packet.GetData(), MockPacketData) {
		return MockAcknowledgement
	}
	return MockFailAcknowledgement
}

func (LitePacketModule) OnAcknowledgementPacket(
	ctx context.Context,
	version string,
	packet exported.PacketI,
	acknowledgement []byte,
	relayer string,
) error {
	return nil
}

func (LitePacketModule) OnTimeoutPacket(
	ctx context.Context,
	version string,
	packet exported.PacketI,
	relayer string,
) error {
	return nil
}
