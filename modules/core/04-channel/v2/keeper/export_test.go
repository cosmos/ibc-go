package keeper

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

import (
	"context"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

func (k *Keeper) SendPacketTest(
	ctx context.Context,
	sourceChannel string,
	timeoutTimestamp uint64,
	payloads []types.Payload,
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
	packet types.Packet,
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
	packet types.Packet,
	acknowledgement types.Acknowledgement,
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
	packet types.Packet,
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

// AliasV1Channel is a wrapper around aliasV1Channel to allow its usage in tests.
func (k *Keeper) AliasV1Channel(ctx context.Context, portID, channelID string) (types.Channel, bool) {
	return k.aliasV1Channel(ctx, portID, channelID)
}
