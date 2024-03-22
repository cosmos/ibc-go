package celestia

import (
	fmt "fmt"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

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
