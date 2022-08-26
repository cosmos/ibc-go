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
		Use:                        "ica controller",
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
		Use: "register",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			connectionID, err := cmd.Flags().GetString(flagConnectionID)
			if err != nil {
				return err
			}
			version, err := cmd.Flags().GetString(flagVersion)
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterAccount(
				connectionID,
				clientCtx.GetFromAddress().String(),
				version,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagConnectionID, "", "Connection ID")
	cmd.Flags().String(flagVersion, "", "Version")
	_ = cmd.MarkFlagRequired(flagConnectionID)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
