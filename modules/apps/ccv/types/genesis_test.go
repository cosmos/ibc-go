package types_test

import (
	"crypto/sha256"
	"testing"
	"time"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/stretchr/testify/require"
)

const (
	chainID                      = "gaia"
	trustingPeriod time.Duration = time.Hour * 24 * 7 * 2
	ubdPeriod      time.Duration = time.Hour * 24 * 7 * 3
	maxClockDrift  time.Duration = time.Second * 10
)

var (
	height      = clienttypes.NewHeight(0, 4)
	upgradePath = []string{"upgrade", "upgradedIBCState"}
)

func TestValidateInitialChildGenesisState(t *testing.T) {
	cs := ibctmtypes.NewClientState(chainID, ibctmtypes.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
	valHash := sha256.Sum256([]byte("mockvalsHash"))
	consensusState := ibctmtypes.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("apphash")), valHash[:])

	cases := []struct {
		name     string
		gs       types.ChildGenesisState
		expError bool
	}{
		{
			"valid new child genesis state",
			types.NewInitialChildGenesisState(cs, consensusState),
			false,
		},
		{
			"invalid new child genesis state: nil client state",
			types.NewInitialChildGenesisState(nil, consensusState),
			true,
		},
		{
			"invalid new child genesis state: invalid client state",
			types.NewInitialChildGenesisState(&ibctmtypes.ClientState{ChainId: "badClientState"}, consensusState),
			true,
		},
		{
			"invalid new child genesis state: nil consensus state",
			types.NewInitialChildGenesisState(cs, nil),
			true,
		},
		{
			"invalid new child genesis state: invalid consensus state",
			types.NewInitialChildGenesisState(cs, &ibctmtypes.ConsensusState{Timestamp: time.Now()}),
			true,
		},
		{
			"invalid new child genesis state: channel id not empty",
			types.ChildGenesisState{
				false,
				"ccvchannel",
				true,
				cs,
				consensusState,
				nil,
			},
			true,
		},
		{
			"invalid new child genesis state: non-nil unbonding sequences",
			types.ChildGenesisState{
				false,
				"",
				true,
				cs,
				consensusState,
				[]types.UnbondingSequence{},
			},
			true,
		},
	}

	for _, c := range cases {
		err := c.gs.Validate()
		if c.expError {
			require.Error(t, err, "%s did not return expected error", c.name)
		} else {
			require.NoError(t, err, "%s returned unexpected error", c.name)
		}
	}
}

func TestValidateRestartGenesisState(t *testing.T) {
	cs := ibctmtypes.NewClientState(chainID, ibctmtypes.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath, false, false)
	valHash := sha256.Sum256([]byte("mockvalsHash"))
	consensusState := ibctmtypes.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("apphash")), valHash[:])
	pk1, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	pk2, err := cryptocodec.ToTmProtoPublicKey(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)

	pd1 := types.NewValidatorSetChangePacketData(
		[]abci.ValidatorUpdate{
			{
				PubKey: pk1,
				Power:  30,
			},
			{
				PubKey: pk2,
				Power:  20,
			},
		},
	)
	pdBytes1, err := pd1.Marshal()
	require.NoError(t, err, "cannot marshal packet data")

	pd2 := types.NewValidatorSetChangePacketData(
		[]abci.ValidatorUpdate{
			{
				PubKey: pk1,
				Power:  40,
			},
			{
				PubKey: pk2,
				Power:  80,
			},
		},
	)
	pdBytes2, err := pd2.Marshal()
	require.NoError(t, err, "cannot marshal packet data")

	cases := []struct {
		name     string
		gs       types.ChildGenesisState
		expError bool
	}{
		{
			"valid restart child genesis state: empty unbonding sequences",
			types.NewRestartChildGenesisState("ccvchannel", nil),
			false,
		},
		{
			"valid restart child genesis state: unbonding sequences",
			types.NewRestartChildGenesisState("ccvchannel", []types.UnbondingSequence{
				types.UnbondingSequence{
					1,
					uint64(time.Now().UnixNano()),
					channeltypes.Packet{
						1, "child", "ccvchannel1",
						"parent", "ccvchannel1",
						pdBytes1,
						clienttypes.NewHeight(0, 100), 0,
					},
				},
				types.UnbondingSequence{
					3,
					uint64(time.Now().UnixNano()),
					channeltypes.Packet{
						3, "child", "ccvchannel1",
						"parent", "ccvchannel1",
						pdBytes2,
						clienttypes.NewHeight(1, 200), 0,
					},
				},
				types.UnbondingSequence{
					5,
					uint64(time.Now().UnixNano()),
					channeltypes.Packet{
						5, "child", "ccvchannel2",
						"parent", "ccvchannel2",
						pdBytes1,
						clienttypes.NewHeight(9, 432), 0,
					},
				},
			}),
			false,
		},
		{
			"invalid restart child genesis state: channel id is empty",
			types.NewRestartChildGenesisState("", nil),
			true,
		},
		{
			"invalid restart child genesis state: unbonding sequence packet is invalid",
			types.NewRestartChildGenesisState("ccvchannel", []types.UnbondingSequence{
				types.UnbondingSequence{
					1,
					uint64(time.Now().UnixNano()),
					channeltypes.Packet{
						1, "", "ccvchannel1",
						"parent", "ccvchannel1",
						pdBytes1,
						clienttypes.NewHeight(0, 100), 0,
					},
				},
			}),
			true,
		},
		{
			"invalid restart child genesis state: unbonding sequence time is invalid",
			types.NewRestartChildGenesisState("ccvchannel", []types.UnbondingSequence{
				types.UnbondingSequence{
					1,
					0,
					channeltypes.Packet{
						1, "child", "ccvchannel1",
						"parent", "ccvchannel1",
						pdBytes1,
						clienttypes.NewHeight(0, 100), 0,
					},
				},
			}),
			true,
		},
		{
			"invalid restart child genesis state: unbonding sequence is invalid",
			types.NewRestartChildGenesisState("ccvchannel", []types.UnbondingSequence{
				types.UnbondingSequence{
					8,
					uint64(time.Now().UnixNano()),
					channeltypes.Packet{
						1, "", "ccvchannel1",
						"parent", "ccvchannel1",
						pdBytes1,
						clienttypes.NewHeight(0, 100), 0,
					},
				},
			}),
			true,
		},
		{
			"invalid restart child genesis: client state defined",
			types.ChildGenesisState{
				false,
				"ccvchannel",
				false,
				cs,
				nil,
				nil,
			},
			true,
		},
		{
			"invalid restart child genesis: consensus state defined",
			types.ChildGenesisState{
				false,
				"ccvchannel",
				false,
				nil,
				consensusState,
				nil,
			},
			true,
		},
	}

	for _, c := range cases {
		err := c.gs.Validate()
		if c.expError {
			require.Error(t, err, "%s did not return expected error", c.name)
		} else {
			require.NoError(t, err, "%s returned unexpected error", c.name)
		}
	}
}

func TestValidateParentGenesisState(t *testing.T) {
	testCases := []struct {
		name     string
		genState types.ParentGenesisState
		expPass  bool
	}{
		{
			"valid initializing parent genesis with nil updates",
			types.NewParentGenesisState(
				[]types.ChildState{{"chainid-1", "channelid", 1}},
			),
			true,
		},
		{
			"valid validating parent genesis with nil updates",
			types.NewParentGenesisState(
				[]types.ChildState{{"chainid-1", "channelid", 2}},
			),
			true,
		},
		{
			"valid multiple parent genesis with multiple child chains",
			types.NewParentGenesisState(
				[]types.ChildState{
					{"chainid-1", "channelid", 2},
					{"chainid-2", "channelid2", 1},
					{"chainid-3", "channelid3", 0},
					{"chainid-4", "channelid4", 3},
				},
			),
			true,
		},
		{
			"invalid chain id",
			types.NewParentGenesisState(
				[]types.ChildState{{"invalidid{}", "channelid", 2}},
			),
			false,
		},
		{
			"invalid channel id",
			types.NewParentGenesisState(
				[]types.ChildState{{"chainid", "invalidchannel{}", 2}},
			),
			false,
		},
	}

	for _, tc := range testCases {
		err := tc.genState.Validate()

		if tc.expPass {
			require.NoError(t, err, "test case: %s must pass", tc.name)
		} else {
			require.Error(t, err, "test case: %s must fail", tc.name)
		}
	}
}
