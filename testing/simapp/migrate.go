package simapp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmtypes "github.com/tendermint/tendermint/types"

	ibctransfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
)

// MigrateGenesisCmd returns a command to execute genesis state migration.
func MigrateGenesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [genesis-file]",
		Short: "Migrate genesis",
		Long: fmt.Sprintf(`Migrate the source genesis and print to STDOUT.

Example:
$ %s migrate /path/to/genesis.json
`, version.AppName),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			var err error

			importGenesis := args[0]

			jsonBlob, err := ioutil.ReadFile(importGenesis)

			if err != nil {
				return errors.Wrap(err, "failed to read provided genesis file")
			}

			genDoc, err := tmtypes.GenesisDocFromJSON(jsonBlob)
			if err != nil {
				return errors.Wrapf(err, "failed to read genesis document from file %s", importGenesis)
			}

			var initialState types.AppMap
			if err := json.Unmarshal(genDoc.AppState, &initialState); err != nil {
				return errors.Wrap(err, "failed to JSON unmarshal initial genesis state")
			}

			newGenState, err := migrateGenesisSlashedDenomsUpgrade(initialState, clientCtx, genDoc)
			if err != nil {
				return errors.Wrap(err, "failed to migrate")
			}

			genDoc.AppState, err = json.Marshal(newGenState)
			if err != nil {
				return errors.Wrap(err, "failed to JSON marshal migrated genesis state")
			}

			bz, err := tmjson.Marshal(genDoc)
			if err != nil {
				return errors.Wrap(err, "failed to marshal genesis doc")
			}

			sortedBz, err := sdk.SortJSON(bz)
			if err != nil {
				return errors.Wrap(err, "failed to sort JSON genesis doc")
			}

			fmt.Println(string(sortedBz))
			return nil
		},
	}

	return cmd
}

// migrateGenesisSlashedDenomsUpgrade corrects any incorrect trace information
// from previously received coins that had slashes in the base denom.
func migrateGenesisSlashedDenomsUpgrade(appState types.AppMap, clientCtx client.Context, genDoc *tmtypes.GenesisDoc) (types.AppMap, error) {
	if appState[ibctransfertypes.ModuleName] != nil {
		transferGenState := &ibctransfertypes.GenesisState{}
		clientCtx.Codec.MustUnmarshalJSON(appState[ibctransfertypes.ModuleName], transferGenState)

		substituteTraces := make([]ibctransfertypes.DenomTrace, len(transferGenState.DenomTraces))
		for i, dt := range transferGenState.DenomTraces {
			// replace all previous traces with the latest trace if validation passes
			// note most traces will have same value
			newTrace := ibctransfertypes.ParseDenomTrace(dt.GetFullDenomPath())

			if err := newTrace.Validate(); err != nil {
				substituteTraces[i] = dt
			} else {
				substituteTraces[i] = newTrace
			}
		}

		transferGenState.DenomTraces = substituteTraces

		// delete old genesis state
		delete(appState, ibctransfertypes.ModuleName)

		// set new ibc transfer genesis state
		appState[ibctransfertypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(transferGenState)
	}

	return appState, nil
}
