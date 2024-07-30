package mock

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

func (cs *ClientState) ClientType() string {
	return ModuleName
}

func (cs *ClientState) Validate() error {
	return nil
}

func (cs *ClientState) Initialize(cdc codec.BinaryCodec, clientStore storetypes.KVStore, consensusState ConsensusState) error {
	setClientState(clientStore, cdc, cs)
	setConsensusState(clientStore, cdc, &consensusState, cs.LatestHeight)

	return nil
}

// GetTimestampAtHeight returns the timestamp in nanoseconds of the consensus state at the given height.
func (ClientState) GetTimestampAtHeight(
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
) (uint64, error) {
	// get consensus state at height from clientStore to check for expiry
	consState, found := getConsensusState(clientStore, cdc, height)
	if !found {
		return 0, errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "height (%s)", height)
	}
	return consState.GetTimestamp(), nil
}
