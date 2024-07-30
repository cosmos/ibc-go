package mock

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// VerifyClientMessage checks if the clientMessage is the correct type and verifies the message
func (*ClientState) VerifyClientMessage(clientMsg exported.ClientMessage) error {
	_, ok := clientMsg.(*MockHeader)
	if !ok {
		return errorsmod.Wrapf(ErrInvalidClientMsg, "invalid client message type %T", clientMsg)
	}

	return nil
}

func (cs *ClientState) UpdateState(cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	mockHeader, ok := clientMsg.(*MockHeader)
	if !ok {
		panic("invalid client message type")
	}

	cs.LatestHeight = mockHeader.Height

	consensusState := ConsensusState{
		Timestamp: mockHeader.Timestamp,
	}

	setClientState(clientStore, cdc, cs)
	setConsensusState(clientStore, cdc, &consensusState, cs.LatestHeight)

	return []exported.Height{cs.LatestHeight}
}
