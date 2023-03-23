package cli

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
	"github.com/spf13/cobra"
)

// getCmdCode defines the command to query wasm code for given code id
func getCmdCode() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "code [code-id]",
		Short: "Query wasm code",
		Long:  "Query wasm code",
		Example: fmt.Sprintf(
			"%s query %s %s code [code-id]", version.AppName, host.SubModuleName, types.ModuleName,
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			codeID := args[0]
			req := types.WasmCodeQuery{
				CodeId: codeID,
			}

			res, err := queryClient.WasmCode(context.Background(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// getCmdCode defines the command to query wasm code for given code id
func getAllWasmCode() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-wasm-code",
		Short: "Query all wasm code",
		Long:  "Query all wasm code",
		Example: fmt.Sprintf(
			"%s query %s all-wasm-code", version.AppName, types.ModuleName,
		),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			req := types.AllWasmCodeIDQuery{
				Pagination: pageReq,
			}

			res, err := queryClient.AllWasmCodeID(context.Background(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "all wasm code")

	return cmd
}
