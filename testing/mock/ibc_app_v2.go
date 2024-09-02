package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// IBCAppV2 contains IBCModuleV2 application module callbacks as defined in 05-port.
type IBCAppV2 struct {
	OnSendPacketV2 func(
		ctx sdk.Context,
		portID string,
		channelID string,
		sequence uint64,
		timeoutHeight clienttypes.Height,
		timeoutTimestamp uint64,
		payload channeltypes.Payload,
		signer sdk.AccAddress,
	) error

	OnRecvPacketV2 func(
		ctx sdk.Context,
		packet channeltypes.PacketV2,
		payload channeltypes.Payload,
		relayer sdk.AccAddress,
	) channeltypes.RecvPacketResult

	OnAcknowledgementPacketV2 func(
		ctx sdk.Context,
		packet channeltypes.PacketV2,
		payload channeltypes.Payload,
		recvPacketResult channeltypes.RecvPacketResult,
		relayer sdk.AccAddress,
	) error

	OnTimeoutPacketV2 func(
		ctx sdk.Context,
		packet channeltypes.PacketV2,
		payload channeltypes.Payload,
		relayer sdk.AccAddress,
	) error
}

// NewIBCV2App returns a IBCAppV2.
func NewIBCV2App() *IBCAppV2 {
	return &IBCAppV2{}
}
