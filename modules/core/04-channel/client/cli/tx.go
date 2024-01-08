package cli

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

const (
	flagMetadata    = "metadata"
	flagSummary     = "summary"
	flagTitle       = "title"
	flagJSON        = "json"
	flagPortPattern = "port-pattern"
	flagExpedited   = "expedited"
)

func newUpgradeChannelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "upgrade-channels",
		Short:   "TODO",
		Long:    "TODO",
		Args:    cobra.ExactArgs(2),
		Example: fmt.Sprintf(`%s tx %s %s upgrade-channels 10stake "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}"`, version.AppName, ibcexported.ModuleName, types.SubModuleName),
		RunE: func(cmd *cobra.Command, args []string) error {
			depositStr := args[0]
			versionStr := args[1]

			metadata, err := cmd.Flags().GetString(flagMetadata)
			if err != nil {
				return err
			}
			summary, err := cmd.Flags().GetString(flagSummary)
			if err != nil {
				return err
			}
			title, err := cmd.Flags().GetString(flagTitle)
			if err != nil {
				return err
			}
			portPattern, err := cmd.Flags().GetString(flagPortPattern)
			if err != nil {
				return err
			}
			displayJSON, err := cmd.Flags().GetBool(flagJSON)
			if err != nil {
				return err
			}
			expidited, err := cmd.Flags().GetBool(flagExpedited)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			resp, err := queryClient.Channels(cmd.Context(), &types.QueryChannelsRequest{
				Pagination: nil,
			})
			if err != nil {
				return err
			}

			var msgs []sdk.Msg
			pattern := regexp.MustCompile(portPattern)

			for _, ch := range resp.Channels {
				// skip any channel that is not open
				if ch.State != types.OPEN {
					continue
				}

				// if the port ID does not match the desired pattern, we skip it.
				if !pattern.MatchString(ch.PortId) {
					continue
				}

				msgUpgradeInit := types.NewMsgChannelUpgradeInit(ch.PortId, ch.ChannelId, types.NewUpgradeFields(ch.Ordering, ch.ConnectionHops, versionStr), clientCtx.GetFromAddress().String())
				msgs = append(msgs, msgUpgradeInit)
			}

			if len(msgs) == 0 {
				return fmt.Errorf("no channels would be upgraded with pattern %s", portPattern)
			}

			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msgSubmitProposal, err := govtypesv1.NewMsgSubmitProposal(msgs, deposit, clientCtx.GetFromAddress().String(), metadata, title, summary, expidited)
			if err != nil {
				return fmt.Errorf("invalid message: %w", err)
			}

			dryRun, _ := cmd.Flags().GetBool(flags.FlagDryRun)
			if displayJSON || dryRun {
				out, err := clientCtx.Codec.MarshalJSON(msgSubmitProposal)
				if err != nil {
					return err
				}
				return clientCtx.PrintBytes(out)
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgSubmitProposal)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String(flagSummary, "Upgrading open channels", "The summary for the gov proposal which will upgrade existing open channels.")
	cmd.Flags().String(flagTitle, "Channel upgrades", "The title for the gov proposal which will upgrade existing open channels.")
	cmd.Flags().String(flagMetadata, "", "Metadata for the gov proposal which will upgrade existing open channels.")
	cmd.Flags().String(flagPortPattern, "transfer", "The pattern to use to match port ids.")
	cmd.Flags().Bool(flagExpedited, false, "set the expedited value for the governance proposal.")
	cmd.Flags().Bool(flagJSON, false, "specify true to output valid proposal.json contents, instead of submitting a governance proposal.")

	return cmd
}
