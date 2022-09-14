package cli

import (
	"io/ioutil"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/types"
	"github.com/spf13/cobra"
)

// NewPushNewWasmCodeCmd returns the command to create a PushNewWasmCode transaction
func NewPushNewWasmCodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push-wasm [wasm-file]",
		Short: "Reads wasm code from the file and creates push transaction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			fileName := args[0]

			code, err := ioutil.ReadFile(fileName)
			if err != nil {
				return err
			}

			msg := &types.MsgPushNewWasmCode{
				Code:   code,
				Signer: clientCtx.GetFromAddress().String(),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
