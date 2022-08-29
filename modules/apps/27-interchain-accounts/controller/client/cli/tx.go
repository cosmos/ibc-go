package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
)

const (
	// The connection end identifier on the controller chain
	flagConnectionID = "connection-id"
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
		Short: "Register account via connection end identifier on the controller chain",
		Args:  cobra.ExactArgs(1),
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
