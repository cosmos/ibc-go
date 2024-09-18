package mock

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// IBCAppV2 contains IBCModuleV2 application module callbacks as defined in 05-port.
type IBCAppV2 struct {
	OnSendPacketV2 func(
		ctx context.Context,
		sourceID string,
		sequence uint64,
		timeoutTimestamp uint64,
		payload channeltypes.Payload,
		signer sdk.AccAddress,
	) error

	OnRecvPacketV2 func(
		ctx context.Context,
		packet channeltypes.PacketV2,
		payload channeltypes.Payload,
		relayer sdk.AccAddress,
	) channeltypes.RecvPacketResult

	OnAcknowledgementPacketV2 func(
		ctx context.Context,
		packet channeltypes.PacketV2,
		payload channeltypes.Payload,
		recvPacketResult channeltypes.RecvPacketResult,
		relayer sdk.AccAddress,
	) error

	OnTimeoutPacketV2 func(
		ctx context.Context,
		packet channeltypes.PacketV2,
		payload channeltypes.Payload,
		relayer sdk.AccAddress,
	) error
}

// NewIBCV2App returns a IBCAppV2.
func NewIBCV2App() *IBCAppV2 {
	return &IBCAppV2{}
}
