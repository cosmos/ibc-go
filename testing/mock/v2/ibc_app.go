package mock

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

type IBCApp struct {
	OnSendPacket            func(goCtx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error
	OnRecvPacket            func(goCtx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult
	OnTimeoutPacket         func(goCtx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error
	OnAcknowledgementPacket func(goCtx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, acknowledgement []byte, relayer sdk.AccAddress) error
}
