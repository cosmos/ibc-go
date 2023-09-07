package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"

	types "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// newStoreCodeCmd returns the command to create a MsgStoreCode transaction
func newStoreCodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "store-code [path/to/wasm-file]",
		Short:   "Reads wasm code from the file and creates transaction to store code",
		Long:    "Reads wasm code from the file and creates transaction to store code",
		Example: fmt.Sprintf("%s tx %s wasm [path/to/wasm_file]", version.AppName, ibcexported.ModuleName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			fileName := args[0]

			code, err := os.ReadFile(fileName)
			if err != nil {
				return err
			}

			msg := &types.MsgStoreCode{
				Signer:       clientCtx.GetFromAddress().String(),
				WasmByteCode: code,
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
