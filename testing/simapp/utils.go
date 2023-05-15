package simapp

import (
	"os"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

// SetupSimulation creates the config, db (levelDB), temporary directory and logger for
// the simulation tests. If `FlagEnabledValue` is false it skips the current test.
// Returns error on an invalid db intantiation or temp dir creation.
func SetupSimulation(dirPrefix, dbName string) (simtypes.Config, dbm.DB, string, log.Logger, bool, error) {
	if !FlagEnabledValue {
		return simtypes.Config{}, nil, "", nil, true, nil
	}

	config := NewConfigFromFlags()
	config.ChainID = "simulation-app"

	var logger log.Logger
	if FlagVerboseValue {
		logger = log.TestingLogger()
	} else {
		logger = log.NewNopLogger()
	}

	dir, err := os.MkdirTemp("", dirPrefix)
	if err != nil {
		return simtypes.Config{}, nil, "", nil, false, err
	}

	db, err := dbm.NewDB(dbName, dbm.BackendType(config.DBBackend), dir)
	if err != nil {
		return simtypes.Config{}, nil, "", nil, false, err
	}

	return config, db, dir, logger, false, nil
}
