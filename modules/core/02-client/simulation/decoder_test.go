package simulation_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/types/kv"

	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	"github.com/cosmos/ibc-go/v7/modules/core/02-client/simulation"
	"github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
)

type testClientUnmarshaller struct {
	cdc codec.Codec
}

func (c *testClientUnmarshaller) MustUnmarshalClientState(bz []byte) exported.ClientState {
	return types.MustUnmarshalClientState(c.cdc, bz)
}
func (c *testClientUnmarshaller) MustUnmarshalConsensusState(bz []byte) exported.ConsensusState {
	return types.MustUnmarshalConsensusState(c.cdc, bz)
}

func TestDecodeStore(t *testing.T) {
	cdc := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{}, ibctm.AppModuleBasic{}).Codec
	clientID := "clientidone"

	height := types.NewHeight(0, 10)

	clientState := &ibctm.ClientState{
		FrozenHeight: height,
	}

	consState := &ibctm.ConsensusState{
		Timestamp: time.Now().UTC(),
	}

	kvPairs := kv.Pairs{
		Pairs: []kv.Pair{
			{
				Key:   host.FullClientStateKey(clientID),
				Value: types.MustMarshalClientState(cdc, clientState),
			},
			{
				Key:   host.FullConsensusStateKey(clientID, height),
				Value: types.MustMarshalConsensusState(cdc, consState),
			},
			{
				Key:   []byte{0x99},
				Value: []byte{0x99},
			},
		},
	}
	tests := []struct {
		name        string
		expectedLog string
	}{
		{"ClientState", fmt.Sprintf("ClientState A: %v\nClientState B: %v", clientState, clientState)},
		{"ConsensusState", fmt.Sprintf("ConsensusState A: %v\nConsensusState B: %v", consState, consState)},
		{"other", ""},
	}

	for i, tt := range tests {
		i, tt := i, tt
		t.Run(tt.name, func(t *testing.T) {
			res, found := simulation.NewDecodeStore(&testClientUnmarshaller{cdc: cdc}, kvPairs.Pairs[i], kvPairs.Pairs[i])
			if i == len(tests)-1 {
				require.False(t, found, string(kvPairs.Pairs[i].Key))
				require.Empty(t, res, string(kvPairs.Pairs[i].Key))
			} else {
				require.True(t, found, string(kvPairs.Pairs[i].Key))
				require.Equal(t, tt.expectedLog, res, string(kvPairs.Pairs[i].Key))
			}
		})
	}
}
