package tokenfactory_test

import (
	"testing"

	"github.com/cosmos/sandbox-ledger/x/tokenfactory"
	"github.com/cosmos/sandbox-ledger/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestAutoCLIOptions(t *testing.T) {
	module := tokenfactory.AppModule{}
	opts := module.AutoCLIOptions()

	require.NotNil(t, opts)
	require.NotNil(t, opts.Query)
	require.NotNil(t, opts.Tx)
}

func TestAppModuleName(t *testing.T) {
	module := tokenfactory.AppModuleBasic{}
	require.Equal(t, types.ModuleName, module.Name())
}

func TestAppModuleConsensusVersion(t *testing.T) {
	module := tokenfactory.AppModule{}
	require.Equal(t, uint64(1), module.ConsensusVersion())
}
