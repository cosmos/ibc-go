package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/ibc-go/modules/core/28-wasm/types"
	"github.com/spf13/cobra"
	"io/ioutil"
)

// NewPushNewWASMCodeCmd returns the command to create a PushNewWASMCode transaction
func NewPushNewWASMCodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push_wasm client_type wasm_file",
		Short: "Reads wasm code from the file and creates push transaction",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			clientType := args[0]
			fileName := args[1]

			code, err := ioutil.ReadFile(fileName)
			if err != nil {
				return err
			}

			msg := &types.MsgPushNewWASMCode{
				ClientType: clientType,
				Code: code,
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
