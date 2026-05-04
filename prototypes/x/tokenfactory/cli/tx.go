package cli

import (
	"fmt"

	"github.com/cosmos/sandbox-ledger/x/tokenfactory/types"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetTxCmd returns the transaction commands for the tokenfactory module.
// Only commands that cannot use AutoCLI (due to cosmos.base.v1beta1.Coin
// positional args being incompatible with protov2 reflection) are defined here.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tokenfactory transaction subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdMint(),
		CmdBurn(),
	)

	return cmd
}

// CmdMint returns the cobra command for minting tokens.
func CmdMint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint [address] [amount]",
		Short: "Mint tokens to a specified address",
		Long:  "Mint tokens to a specified address. The sender must be the admin of the token.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			address := args[0]
			coin, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return fmt.Errorf("invalid coin: %w", err)
			}

			msg := types.NewMsgMint(
				clientCtx.GetFromAddress().String(),
				address,
				coin,
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdBurn returns the cobra command for burning tokens.
func CmdBurn() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn [amount]",
		Short: "Burn tokens from the sender's balance",
		Long:  "Burn tokens from the sender's balance. The sender must be the admin of the token.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			coin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return fmt.Errorf("invalid coin: %w", err)
			}

			msg := types.NewMsgBurn(
				clientCtx.GetFromAddress().String(),
				coin,
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
