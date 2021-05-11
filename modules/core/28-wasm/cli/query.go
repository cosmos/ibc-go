package cli

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/28-wasm/types"
	"github.com/spf13/cobra"
)


// GetCmdQueryLatestWASMCode defines the command to query latest wasm code
// uploaded for that client type
func GetCmdQueryLatestWASMCode() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wasm_code client_type",
		Short:   "Query latest wasm code",
		Long:    "Query latest wasm code for particular client type",
		Example: fmt.Sprintf("%s query %s %s wasm_code client_type", version.AppName, host.ModuleName, types.SubModuleName),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			clientType := args[0]
			req := types.LatestWASMCodeQuery{
				ClientType: clientType,
			}

			res, err := queryClient.LatestWASMCode(context.Background(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryLatestWASMCodeEntry defines the command to query latest wasm code's entry
// for that client type
func GetCmdQueryLatestWASMCodeEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wasm_code_entry client_type",
		Short: "Query wasm code entry",
		Long:  "Query latest wasm code entry",
		Example: fmt.Sprintf(
			"%s query %s %s wasm_code_entry client_type", version.AppName, host.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			clientType := args[0]
			req := types.LatestWASMCodeEntryQuery{
				ClientType: clientType,
			}

			res, err := queryClient.LatestWASMCodeEntry(context.Background(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
