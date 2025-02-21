package keeper

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

func (k *Keeper) SendPacketTest(
	ctx sdk.Context,
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
	ctx sdk.Context,
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
	ctx sdk.Context,
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
	ctx sdk.Context,
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
