package types_test

import (
	fmt "fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func TestValidateBasic(t *testing.T) {
	var (
		proposal govtypes.Content
		err      error
	)
	clientState := ibctmtypes.NewClientState(
		"chainID", ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift,
		clienttypes.NewHeight(0, 1), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, true, true,
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {
				proposal, err = types.NewCreateChildChainProposal("title", "description", "chainID", clientState, []byte("gen_hash"), time.Now())
				require.NoError(t, err)
			}, true,
		},
		{
			"fails validate abstract - empty title", func() {
				proposal, err = types.NewCreateChildChainProposal(" ", "description", "chainID", clientState, []byte("gen_hash"), time.Now())
				require.NoError(t, err)
			}, false,
		},
		{
			"chainID is empty", func() {
				proposal, err = types.NewCreateChildChainProposal("title", "description", " ", clientState, []byte("gen_hash"), time.Now())
				require.NoError(t, err)
			}, false,
		},
		{
			"clientstate is nil", func() {
				proposal = &types.CreateChildChainProposal{
					Title:       "title",
					Description: "description",
					ChainId:     "chainID",
					ClientState: nil,
					GenesisHash: []byte("gen_hash"),
					SpawnTime:   time.Now(),
				}
			}, false,
		},
		{
			"clientstate cannot be unpacked", func() {
				any, err := clienttypes.PackConsensusState(&ibctmtypes.ConsensusState{})
				require.NoError(t, err)

				proposal = &types.CreateChildChainProposal{
					Title:       "title",
					Description: "description",
					ChainId:     "chainID",
					ClientState: any,
					GenesisHash: []byte("gen_hash"),
					SpawnTime:   time.Now(),
				}
			}, false,
		},
		{
			"genesis hash is empty", func() {
				proposal, err = types.NewCreateChildChainProposal("title", "description", "chainID", clientState, []byte(""), time.Now())
				require.NoError(t, err)
			}, false,
		},
		{
			"time is zero", func() {
				proposal, err = types.NewCreateChildChainProposal("title", "description", "chainID", clientState, []byte("gen_hash"), time.Time{})
				require.NoError(t, err)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		err := proposal.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid case: %s should not return error. got %w", tc.name, err)
		} else {
			require.Error(t, err, "invalid case: %s must return error but got none")
		}
	}
}

func TestMarshalCreateChildChainProposal(t *testing.T) {
	clientState := ibctmtypes.NewClientState(
		"chainID", ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift,
		clienttypes.NewHeight(0, 1), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, true, true,
	)
	content, err := types.NewCreateChildChainProposal("title", "description", "chainID", clientState, []byte("gen_hash"), time.Now().UTC())
	require.NoError(t, err)

	cccp, ok := content.(*types.CreateChildChainProposal)
	require.True(t, ok)

	// create codec
	ir := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(ir)
	govtypes.RegisterInterfaces(ir)
	clienttypes.RegisterInterfaces(ir)
	ibctmtypes.RegisterInterfaces(ir)
	cdc := codec.NewProtoCodec(ir)

	// marshal proposal
	bz, err := cdc.MarshalJSON(cccp)
	require.NoError(t, err)

	// unmarshal proposal
	newCccp := &types.CreateChildChainProposal{}
	err = cdc.UnmarshalJSON(bz, newCccp)
	require.NoError(t, err)

	_, err = clienttypes.UnpackClientState(newCccp.ClientState)
	require.NoError(t, err)

	require.True(t, proto.Equal(cccp, newCccp), "unmarshalled proposal does not equal original proposal")
}

func TestCreateChildChainProposalString(t *testing.T) {
	clientState := ibctmtypes.NewClientState(
		"chainID", ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift,
		clienttypes.NewHeight(0, 1), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, true, true,
	)
	spawnTime := time.Now()
	proposal, err := types.NewCreateChildChainProposal("title", "description", "chainID", clientState, []byte("gen_hash"), spawnTime)
	require.NoError(t, err)

	expect := fmt.Sprintf(`CreateChildChain Proposal
	Title: title
	Description: description
	ChainID: chainID
	ClientState: %s
	GenesisHash: %s
	SpawnTime: %s`, clientState.String(), []byte("gen_hash"), spawnTime)

	require.Equal(t, expect, proposal.String(), "string method for CreateChildChainProposal returned unexpected string")
}
