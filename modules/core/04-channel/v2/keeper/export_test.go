package keeper

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

import (
	"context"

	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

func (k *Keeper) SendPacketTest(
	ctx context.Context,
	sourceChannel string,
	timeoutTimestamp uint64,
	payloads []channeltypesv2.Payload,
) (uint64, string, error) {
	return k.sendPacket(
		ctx,
		sourceChannel,
		timeoutTimestamp,
		payloads,
	)
}

func (k *Keeper) RecvPacketTest(
	ctx context.Context,
	packet channeltypesv2.Packet,
	proof []byte,
	proofHeight exported.Height,
) error {
	return k.recvPacket(
		ctx,
		packet,
		proof,
		proofHeight,
	)
}

func (k *Keeper) AcknowledgePacketTest(
	ctx context.Context,
	packet channeltypesv2.Packet,
	acknowledgement channeltypesv2.Acknowledgement,
	proof []byte,
	proofHeight exported.Height,
) error {
	return k.acknowledgePacket(
		ctx,
		packet,
		acknowledgement,
		proof,
		proofHeight,
	)
}

func (k *Keeper) TimeoutPacketTest(
	ctx context.Context,
	packet channeltypesv2.Packet,
	proof []byte,
	proofHeight exported.Height,
) error {
	return k.timeoutPacket(
		ctx,
		packet,
		proof,
		proofHeight,
	)
}
