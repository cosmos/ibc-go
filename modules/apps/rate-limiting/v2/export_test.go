package v2

import (
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
)

func V2ToV1Packet(payload channeltypesv2.Payload, sourceClient, destinationClient string, sequence uint64) (channeltypes.Packet, error) {
	return v2ToV1Packet(payload, sourceClient, destinationClient, sequence)
}
