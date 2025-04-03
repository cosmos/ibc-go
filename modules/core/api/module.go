package api

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

// IBCModule defines an interface that implements all the callbacks
// that modules must define as specified in IBC Protocol V2.
type IBCModule interface {
	// OnSendPacket is executed when a packet is being sent from sending chain.
	// this callback is provided with the source and destination IDs, the signer, the packet sequence and the packet data
	// for this specific application.
	OnSendPacket(
		ctx sdk.Context,
		sourceClient string,
		destinationClient string,
		sequence uint64,
		payload channeltypesv2.Payload,
		signer sdk.AccAddress,
	) error

	OnRecvPacket(
		ctx sdk.Context,
		sourceClient string,
		destinationClient string,
		sequence uint64,
		payload channeltypesv2.Payload,
		relayer sdk.AccAddress,
	) channeltypesv2.RecvPacketResult

	// OnTimeoutPacket is executed when a packet has timed out on the receiving chain.
	OnTimeoutPacket(
		ctx sdk.Context,
		sourceClient string,
		destinationClient string,
		sequence uint64,
		payload channeltypesv2.Payload,
		relayer sdk.AccAddress,
	) error

	// OnAcknowledgementPacket is executed when a packet gets acknowledged
	OnAcknowledgementPacket(
		ctx sdk.Context,
		sourceClient string,
		destinationClient string,
		sequence uint64,
		acknowledgement []byte,
		payload channeltypesv2.Payload,
		relayer sdk.AccAddress,
	) error
}

type WriteAcknowledgementWrapper interface {
	// WriteAcknowledgement writes the acknowledgement for an async acknowledgement
	WriteAcknowledgement(
		ctx sdk.Context,
		srcClientID string,
		sequence uint64,
		ack channeltypesv2.Acknowledgement,
	) error
}

// PacketDataUnmarshaler defines an optional interface which allows a middleware
// to request the packet data to be unmarshaled by the base application.
type PacketDataUnmarshaler interface {
	// UnmarshalPacketData unmarshals the packet data into a concrete type
	// the payload is provided and the packet data interface is returned
	UnmarshalPacketData(payload channeltypesv2.Payload) (any, error)
}
