package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"regexp"
	"slices"
	"strings"

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
	flagChannelIDs  = "channel-ids"
)

func newUpgradeChannelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade-channels",
		Short: "Upgrade IBC channels",
		Long: `Submit a governance proposal to upgrade all open channels whose port matches a specified pattern 
(the default is transfer), optionally, specific an exact list of channel IDs with a comma separated list.`,
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

			commaSeparatedChannelIDs, err := cmd.Flags().GetString(flagChannelIDs)
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

			resp, err := queryClient.Channels(cmd.Context(), &types.QueryChannelsRequest{})
			if err != nil {
				return err
			}

			channelIDs := strings.Split(commaSeparatedChannelIDs, ",")

			var msgs []sdk.Msg
			pattern := regexp.MustCompile(portPattern)

			for _, ch := range resp.Channels {

				if !channelShouldBeUpgraded(*ch, pattern, channelIDs) {
					continue
				}

				// construct a MsgChannelUpgradeInit which will upgrade the specified channel to a specific version.
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
	cmd.Flags().String(flagChannelIDs, "", "a comma separated list of channel IDs to upgrade.")

	return cmd
}

// channelShouldBeUpgraded returns a boolean indicated whether or not the given channel should be upgraded based
// on either the provided regex pattern or list of desired channel IDs.
func channelShouldBeUpgraded(channel types.IdentifiedChannel, pattern *regexp.Regexp, channelIDs []string) bool {
	// skip any channel that is not open
	if channel.State != types.OPEN {
		return false
	}

	// if specified, the channel ID must exactly match.
	if len(channelIDs) > 0 {
		return pattern.MatchString(channel.PortId) && slices.Contains(channelIDs, channel.ChannelId)
	}

	// otherwise we only need the port pattern to match.
	return pattern.MatchString(channel.PortId)
}
