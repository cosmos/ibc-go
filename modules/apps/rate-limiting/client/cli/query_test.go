// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/client/cli"
)

func TestQueryRateLimitFlagGroups(t *testing.T) {
	paginationArgs := map[string]string{
		flags.FlagPage:       "--page=2",
		flags.FlagPageKey:    "--page-key=key",
		flags.FlagOffset:     "--offset=1",
		flags.FlagLimit:      "--limit=10",
		flags.FlagCountTotal: "--count-total",
		flags.FlagReverse:    "--reverse",
	}

	for flagName, arg := range paginationArgs {
		t.Run("denom_excludes_"+flagName, func(t *testing.T) {
			cmd := cli.GetCmdQueryRateLimit()
			require.NoError(t, cmd.ParseFlags([]string{"--denom=uatom", arg}))
			require.ErrorContains(t, cmd.ValidateFlagGroups(),
				"if any flags in the group [denom "+flagName+"] are set none of the others can be")
		})
	}

	t.Run("pagination_flags_combine", func(t *testing.T) {
		cmd := cli.GetCmdQueryRateLimit()
		args := make([]string, 0, len(paginationArgs))
		for _, arg := range paginationArgs {
			args = append(args, arg)
		}
		require.NoError(t, cmd.ParseFlags(args))
		require.NoError(t, cmd.ValidateFlagGroups())
	})

	t.Run("denom_alone", func(t *testing.T) {
		cmd := cli.GetCmdQueryRateLimit()
		require.NoError(t, cmd.ParseFlags([]string{"--denom=uatom"}))
		require.NoError(t, cmd.ValidateFlagGroups())
	})
}
