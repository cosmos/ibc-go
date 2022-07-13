package cmd_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v3/testing/simapp"
	"github.com/cosmos/ibc-go/v3/testing/simapp/simd/cmd"
)

func TestInitCmd(t *testing.T) {
	rootCmd, _ := cmd.NewRootCmd()
	rootCmd.SetArgs([]string{
		"init",        // Test the init cmd
		"simapp-test", // Moniker
		fmt.Sprintf("--%s=%s", cli.FlagOverwrite, "true"), // Overwrite genesis.json, in case it already exists
	})

	require.NoError(t, svrcmd.Execute(rootCmd, "", simapp.DefaultNodeHome))
}

func TestHomeFlagRegistration(t *testing.T) {
	homeDir := "/tmp/foo"

	rootCmd, _ := cmd.NewRootCmd()

	rootCmd.SetArgs([]string{
		"query",
		fmt.Sprintf("--%s", flags.FlagHome),
		homeDir,
	})

	require.NoError(t, svrcmd.Execute(rootCmd, "", simapp.DefaultNodeHome))

	result, err := rootCmd.Flags().GetString(flags.FlagHome)
	require.NoError(t, err)
	require.Equal(t, result, homeDir)
}
