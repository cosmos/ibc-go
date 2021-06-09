package v100

import (
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	clientv100 "github.com/cosmos/ibc-go/modules/core/02-client/legacy/v100"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/types"
)

// MigrateGenesis accepts exported v1.0.0 IBC client genesis file and migrates it to:
//
// - Update solo machine client state protobuf definition (v1 to v2)
// - Remove all solo machine consensus states
// - Remove all expired tendermint consensus states
func MigrateGenesis(appState genutiltypes.AppMap, clientCtx client.Context, genesisBlockTime time.Time) (genutiltypes.AppMap, error) {
	if appState[host.ModuleName] != nil {
		// unmarshal relative source genesis application state
		ibcGenState := &types.GenesisState{}
		clientCtx.JSONCodec.MustUnmarshalJSON(appState[host.ModuleName], ibcGenState)

		clientGenState, err := clientv100.MigrateGenesis(codec.NewProtoCodec(clientCtx.InterfaceRegistry), &ibcGenState.ClientGenesis, genesisBlockTime)
		if err != nil {
			return nil, err
		}

		ibcGenState.ClientGenesis = *clientGenState

		// delete old genesis state
		delete(appState, host.ModuleName)

		// set new ibc genesis state
		appState[host.ModuleName] = clientCtx.JSONCodec.MustMarshalJSON(ibcGenState)
	}
	return appState, nil
}
