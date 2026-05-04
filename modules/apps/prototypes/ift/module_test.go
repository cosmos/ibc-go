package ift_test

import (
	"testing"

	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/ift"
	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/ift/types"
	"github.com/stretchr/testify/require"
)

func TestAutoCLIOptions(t *testing.T) {
	module := ift.AppModule{}
	opts := module.AutoCLIOptions()

	require.NotNil(t, opts)
	require.NotNil(t, opts.Query)
	require.NotNil(t, opts.Tx)
}

func TestAppModuleName(t *testing.T) {
	module := ift.AppModule{}
	require.Equal(t, types.ModuleName, module.Name())
}

func TestAppModuleConsensusVersion(t *testing.T) {
	module := ift.AppModule{}
	require.Equal(t, uint64(1), module.ConsensusVersion())
}
