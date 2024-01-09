package cli

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"

	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

const (
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
		Args:    cobra.ExactArgs(1),
		Example: fmt.Sprintf(`%s tx %s %s upgrade-channels "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}" --deposit 10stake`, version.AppName, ibcexported.ModuleName, types.SubModuleName),
		RunE: func(cmd *cobra.Command, args []string) error {
			versionStr := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
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

			queryClient := types.NewQueryClient(clientCtx)

			resp, err := queryClient.Channels(cmd.Context(), &types.QueryChannelsRequest{})
			if err != nil {
				return err
			}

			channelIDs := getChannelIDs(commaSeparatedChannelIDs)

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

			msgSubmitProposal, err := govcli.ReadGovPropFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			if err := msgSubmitProposal.SetMsgs(msgs); err != nil {
				return err
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
	govcli.AddGovPropFlagsToCmd(cmd)
	cmd.Flags().Bool(flagJSON, false, "specify true to output valid proposal.json contents, instead of submitting a governance proposal.")
	cmd.Flags().String(flagPortPattern, "transfer", "The pattern to use to match port ids.")
	cmd.Flags().String(flagChannelIDs, "", "a comma separated list of channel IDs to upgrade.")
	cmd.Flags().Bool(flagExpedited, false, "set the expedited value for the governance proposal.")

	return cmd
}

// getChannelIDs returns a slice of channel IDs based on a comma separated string of channel IDs.
func getChannelIDs(commaSeparatedList string) []string {
	if strings.TrimSpace(commaSeparatedList) == "" {
		return nil
	}
	return strings.Split(commaSeparatedList, ",")
}

// channelShouldBeUpgraded returns a boolean indicating whether or not the given channel should be upgraded based
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
