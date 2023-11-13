package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// getCmdCode defines the command to query wasm code for given code hash.
func getCmdCode() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "code [code-hash]",
		Short:   "Query wasm code",
		Long:    "Query wasm code for a light client wasm contract with a given code hash",
		Example: fmt.Sprintf("%s query %s wasm code [code-hash]", version.AppName, ibcexported.ModuleName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			checksum := args[0]
			req := types.QueryCodeRequest{
				Checksum: checksum,
			}

			res, err := queryClient.Code(context.Background(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// getCmdCodeHashes defines the command to query all wasm code hashes.
func getCmdCodeHashes() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "code-hashes",
		Short:   "Query all code hashes",
		Long:    "Query all code hashes for all deployed light client wasm contracts",
		Example: fmt.Sprintf("%s query %s wasm code-hashes", version.AppName, ibcexported.ModuleName),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := types.QueryChecksumsRequest{}

			res, err := queryClient.Checksums(context.Background(), &req)
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
