package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/light-clients/10-wasm/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
)

func NewCreateClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create [CodeID in hex] [path/to/consensus_state.bin] [path/to/client_state.bin]",
		Short:   "create new wasm client",
		Long:    "Create a new wasm IBC client",
		Example: fmt.Sprintf("%s tx ibc %s create [path/to/consensus_state.json] [path/to/client_state.json]", version.AppName, types.SubModuleName),
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			clientStateBytes, err := ioutil.ReadFile(args[1])
			if err != nil {
				return errors.Wrap(err, "error reading client state from file")
			}

			clientState := types.ClientState{}
			if err := json.Unmarshal(clientStateBytes, &clientState); err != nil {
				return errors.Wrap(err, "error unmarshalling client state")
			}

			consensusStateBytes, err := ioutil.ReadFile(args[2])
			if err != nil {
				return errors.Wrap(err, "error reading consensus state from file")
			}

			consensusState := types.ConsensusState{}
			if err := json.Unmarshal(consensusStateBytes, &consensusState); err != nil {
				return errors.Wrap(err, "error unmarshalling consensus state")
			}

			if bytes.Compare(clientState.CodeId, consensusState.CodeId) != 0 {
				return fmt.Errorf("CodeId mismatch between client state and consensus state")
			}

			msg, err := clienttypes.NewMsgCreateClient(
				&clientState, &consensusState, clientCtx.GetFromAddress().String(),
			)
			if err != nil {
				return errors.Wrap(err, "error composing MsgCreateClient")
			}

			if err := msg.ValidateBasic(); err != nil {
				return errors.Wrap(err, "error validating MsgCreateClient")
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func NewUpdateClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [client-id] [Code Id in hex] [path/to/header.bin]",
		Short: "update existing client with a header",
		Long:  "update existing wasm client with a header",
		Example: fmt.Sprintf(
			"$ %s tx ibc %s update [client-id] [path/to/header.json] --from node0 --home ../node0/<app>cli --chain-id $CID",
			version.AppName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			clientID := args[0]

			headerBytes, err := ioutil.ReadFile(args[2])
			if err != nil {
				return errors.Wrap(err, "error reading header from file")
			}

			header := types.Header{}
			if err := json.Unmarshal(headerBytes, &header); err != nil {
				return errors.Wrap(err, "error unmarshalling header")
			}

			msg, err := clienttypes.NewMsgUpdateClient(clientID, &header, clientCtx.GetFromAddress().String())
			if err != nil {
				return errors.Wrap(err, "error composing MsgUpdateClient")
			}

			if err := msg.ValidateBasic(); err != nil {
				return errors.Wrap(err, "error validating MsgUpdateClient")
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func NewSubmitMisbehaviourCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "misbehaviour [client-Id] [path/to/misbehaviour.json]",
		Short: "submit a client misbehaviour",
		Long:  "submit a client misbehaviour to invalidate to invalidate previous state roots and prevent future updates",
		Example: fmt.Sprintf(
			"$ %s tx ibc %s misbehaviour [client-Id] [path/to/misbehaviour.json] --from node0 --home ../node0/<app>cli --chain-id $CID",
			version.AppName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			clientID := args[0]

			misbehaviourBytes, err := ioutil.ReadFile(args[2])
			if err != nil {
				return errors.Wrap(err, "error reading header1 from file")
			}

			misbehaviour := types.Misbehaviour{}
			if err := json.Unmarshal(misbehaviourBytes, &misbehaviour); err != nil {
				return errors.Wrap(err, "error unmarshalling misbehaviour")
			}
			misbehaviour.ClientId = clientID

			msg, err := clienttypes.NewMsgSubmitMisbehaviour(misbehaviour.ClientId, &misbehaviour, clientCtx.GetFromAddress().String())
			if err != nil {
				return errors.Wrap(err, "error composing MsgSubmitMisbehaviour")
			}

			if err := msg.ValidateBasic(); err != nil {
				return errors.Wrap(err, "error validating MsgSubmitMisbehaviour")
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
