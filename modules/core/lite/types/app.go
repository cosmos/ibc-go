package types

import (
	"context"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type PacketModule interface {
	OnSendPacket(
		ctx context.Context,
		sourcePort string,
		sourceChannel string,
		sequence uint64,
		version string,
		data []byte,
		sender string,
	) error

	// OnRecvPacket must return an acknowledgement that implements the Acknowledgement interface.
	// In the case of an asynchronous acknowledgement, nil should be returned.
	// If the acknowledgement returned is successful, the state changes on callback are written,
	// otherwise the application state changes are discarded. In either case the packet is received
	// and the acknowledgement is written (in synchronous cases).
	OnRecvPacket(
		ctx context.Context,
		version string,
		packet exported.PacketI,
		relayer string,
	) exported.Acknowledgement

	OnAcknowledgementPacket(
		ctx context.Context,
		version string,
		packet exported.PacketI,
		acknowledgement []byte,
		relayer string,
	) error

	OnTimeoutPacket(
		ctx context.Context,
		version string,
		packet exported.PacketI,
		relayer string,
	) error
}
