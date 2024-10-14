package api

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// IBCModule defines an interface that implements all the callbacks
// that modules must define as specified in IBC Protocol V2.
type IBCModule interface {
	// OnSendPacket is executed when a packet is being sent from sending chain.
	// this callback is provided with the source and destination IDs, the signer, the packet sequence and the packet data
	// for this specific application.
	OnSendPacket(
		ctx context.Context,
		sourceChannel string,
		destinationChannel string,
		sequence uint64,
		data channeltypesv2.PacketData,
		signer sdk.AccAddress,
	) error

	OnRecvPacket(
		ctx context.Context,
		sourceChannel string,
		destinationChannel string,
		data channeltypesv2.PacketData,
		relayer sdk.AccAddress,
	) channeltypesv2.RecvPacketResult

	// OnTimeoutPacket is executed when a packet has timed out on the receiving chain.
	OnTimeoutPacket(
		ctx context.Context,
		sourceChannel string,
		destinationChannel string,
		data channeltypesv2.PacketData,
		relayer sdk.AccAddress,
	) error

	// OnAcknowledgementPacket is executed when a packet gets acknowledged
	OnAcknowledgementPacket(
		ctx context.Context,
		sourceChannel string,
		destinationChannel string,
		data channeltypesv2.PacketData,
		acknowledgement []byte,
		relayer sdk.AccAddress,
	) error
}
