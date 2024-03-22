package celestia

import (
	fmt "fmt"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

// VerifyClientMessage implements exported.ClientState.
func (cs *ClientState) VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) error {
	return cs.BaseClient.VerifyClientMessage(ctx, cdc, clientStore, clientMsg)
}

// UpdateState implements exported.ClientState.
func (cs *ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	header, ok := clientMsg.(*ibctm.Header)
	if !ok {
		panic(fmt.Errorf("expected type %T, got %T", &ibctm.Header{}, clientMsg))
	}

	// perform regular 07-tendermint client update step
	heights := cs.BaseClient.UpdateState(ctx, cdc, clientStore, clientMsg)

	// overwrite the consensus state with a new consensus state containing the data hash as commitment root
	consensusState := &ibctm.ConsensusState{
		Timestamp:          header.GetTime(),
		Root:               commitmenttypes.NewMerkleRoot(header.Header.GetDataHash()),
		NextValidatorsHash: header.Header.NextValidatorsHash,
	}

	setConsensusState(clientStore, cdc, consensusState, header.GetHeight())

	return heights
}

// setConsensusState stores the consensus state at the given height.
func setConsensusState(clientStore storetypes.KVStore, cdc codec.BinaryCodec, consensusState *ibctm.ConsensusState, height exported.Height) {
	key := host.ConsensusStateKey(height)
	val := clienttypes.MustMarshalConsensusState(cdc, consensusState)
	clientStore.Set(key, val)
}

// CheckForMisbehaviour implements exported.ClientState.
func (cs *ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) bool {
	return cs.BaseClient.CheckForMisbehaviour(ctx, cdc, clientStore, clientMsg)
}

// UpdateStateOnMisbehaviour implements exported.ClientState.
func (cs *ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) {
	cs.BaseClient.UpdateStateOnMisbehaviour(ctx, cdc, clientStore, clientMsg)
}
