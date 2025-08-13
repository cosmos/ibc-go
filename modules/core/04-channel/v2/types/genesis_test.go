package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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
				[]types.PacketState{types.NewPacketState(ibctesting.FirstChannelID, 1, []byte("ack"))},
				[]types.PacketState{types.NewPacketState(ibctesting.SecondChannelID, 1, []byte(""))},
				[]types.PacketState{types.NewPacketState(ibctesting.FirstChannelID, 1, []byte("commit_hash"))},
				[]types.PacketState{types.NewPacketState(ibctesting.SecondChannelID, 1, []byte("async_packet"))},
				[]types.PacketSequence{types.NewPacketSequence(ibctesting.SecondChannelID, 1)},
			),
			nil,
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
			"invalid async packet",
			types.GenesisState{
				AsyncPackets: []types.PacketState{
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
