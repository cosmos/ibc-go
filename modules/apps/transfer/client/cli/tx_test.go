// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v11/modules/apps/transfer/client/cli"
)

func TestTransferRejectsMissingArguments(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		err  string
	}{
		{"missing receiver and coin", []string{"transfer", "channel-0"}, "accepts 4 arg(s), received 2"},
		{"missing coin", []string{"transfer", "channel-0", "receiver"}, "accepts 4 arg(s), received 3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := cli.NewTransferTxCmd()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)

			require.NotPanics(t, func() {
				err := cmd.ExecuteContext(t.Context())
				require.EqualError(t, err, tc.err)
			})
		})
	}
}

func TestTransferAcceptsAllArguments(t *testing.T) {
	cmd := cli.NewTransferTxCmd()
	require.NoError(t, cmd.ValidateArgs([]string{"transfer", "channel-0", "receiver", "1stake"}))
}
