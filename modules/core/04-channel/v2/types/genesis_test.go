package types_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func TestValidateGenesis(t *testing.T) {
	testCases := []struct {
		name     string
		genState types.GenesisState
		expError error
	}{
		{
			"default",
			types.DefaultGenesisState(),
			nil,
		},
		{
			"valid genesis",
			types.NewGenesisState(
				[]types.IdentifiedChannel{
					types.NewIdentifiedChannel(
						ibctesting.FirstChannelID, types.NewChannel(ibctesting.FirstClientID, ibctesting.SecondChannelID, ibctesting.MerklePath),
					),
					types.NewIdentifiedChannel(
						ibctesting.SecondChannelID, types.NewChannel(ibctesting.SecondClientID, ibctesting.FirstChannelID, ibctesting.MerklePath),
					),
				},
				[]types.PacketState{types.NewPacketState(ibctesting.FirstChannelID, 1, []byte("ack"))},
				[]types.PacketState{types.NewPacketState(ibctesting.SecondChannelID, 1, []byte(""))},
				[]types.PacketState{types.NewPacketState(ibctesting.FirstChannelID, 1, []byte("commit_hash"))},
				[]types.PacketSequence{types.NewPacketSequence(ibctesting.SecondChannelID, 1)},
				2,
			),
			nil,
		},
		{
			"invalid channel identifier",
			types.GenesisState{
				Channels: []types.IdentifiedChannel{types.NewIdentifiedChannel(ibctesting.InvalidID, types.NewChannel(ibctesting.FirstClientID, ibctesting.SecondChannelID, ibctesting.MerklePath))},
			},
			host.ErrInvalidID,
		},
		{
			"invalid ack",
			types.GenesisState{
				Acknowledgements: []types.PacketState{
					types.NewPacketState(ibctesting.SecondChannelID, 1, nil),
				},
			},
			errors.New("data bytes cannot be nil"),
		},
		{
			"invalid commitment",
			types.GenesisState{
				Commitments: []types.PacketState{
					types.NewPacketState(ibctesting.FirstChannelID, 1, nil),
				},
			},
			errors.New("data bytes cannot be nil"),
		},
		{
			"invalid send seq",
			types.GenesisState{
				SendSequences: []types.PacketSequence{
					types.NewPacketSequence(ibctesting.FirstChannelID, 0),
				},
			},
			errors.New("sequence cannot be 0"),
		},
		{
			"next channel sequence is less than maximum channel identifier sequence used",
			types.GenesisState{
				Channels: []types.IdentifiedChannel{
					types.NewIdentifiedChannel("channel-10", types.NewChannel(ibctesting.FirstClientID, ibctesting.SecondChannelID, ibctesting.MerklePath)),
				},
				NextChannelSequence: 0,
			},
			fmt.Errorf("next channel sequence 0 must be greater than maximum sequence used in channel identifier 10"),
		},
	}

	for _, tc := range testCases {

		err := tc.genState.Validate()

		expPass := tc.expError == nil
		if expPass {
			require.NoError(t, err)
		} else {
			ibctesting.RequireErrorIsOrContains(t, err, tc.expError)
		}
	}
}
