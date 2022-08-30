package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"strings"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
)

const (
	// The controller chain channel version
	flagVersion = "version"
)

// NewTxCmd creates and returns the tx command
func NewTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "controller",
		Short:                      "ica controller transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		newRegisterAccountCmd(),
	)

	return cmd
}

func newRegisterAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [connection-id]",
		Short: "Register an interchain account on the provided connection.",
		Long: strings.TrimSpace(`Register an account on the counterparty chain via the 
connection id from the source chain. Connection identifier should be for the source chain 
and that the account will be created on the counterparty chain. Callers are expected to 
provide the appropriate application version string via {version} flag. Generates a new 
port identifier using the provided owner string, binds to the port identifier and claims 
the associated capability.`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			connectionID := args[0]
			owner := clientCtx.GetFromAddress().String()
			version, err := cmd.Flags().GetString(flagVersion)
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterAccount(connectionID, owner, version)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagVersion, "", "Controller chain channel version")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
