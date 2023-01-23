package types_test

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

func (suite *TypesTestSuite) TestValidateBasic() {
	subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupClients(subjectPath)
	subject := subjectPath.EndpointA.ClientID

	substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupClients(substitutePath)
	substitute := substitutePath.EndpointA.ClientID

	testCases := []struct {
		name     string
		proposal govtypes.Content
		expPass  bool
	}{
		{
			"success",
			types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute),
			true,
		},
		{
			"fails validate abstract - empty title",
			types.NewClientUpdateProposal("", ibctesting.Description, subject, substitute),
			false,
		},
		{
			"subject and substitute use the same identifier",
			types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, subject),
			false,
		},
		{
			"invalid subject clientID",
			types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, ibctesting.InvalidID, substitute),
			false,
		},
		{
			"invalid substitute clientID",
			types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, ibctesting.InvalidID),
			false,
		},
	}

	for _, tc := range testCases {

		err := tc.proposal.ValidateBasic()

		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}

// tests a client update proposal can be marshaled and unmarshaled
func (suite *TypesTestSuite) TestMarshalClientUpdateProposalProposal() {
	// create proposal
	proposal := types.NewClientUpdateProposal("update IBC client", "description", "subject", "substitute")

	// create codec
	ir := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(ir)
	govtypes.RegisterInterfaces(ir)
	cdc := codec.NewProtoCodec(ir)

	// marshal message
	content := proposal.(*types.ClientUpdateProposal)
	bz, err := cdc.MarshalJSON(content)
	suite.Require().NoError(err)

	// unmarshal proposal
	newProposal := &types.ClientUpdateProposal{}
	err = cdc.UnmarshalJSON(bz, newProposal)
	suite.Require().NoError(err)
}

func (suite *TypesTestSuite) TestUpgradeProposalValidateBasic() {
	var (
		proposal govtypes.Content
		err      error
	)

	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupClients(path)
	cs := suite.chainA.GetClientState(path.EndpointA.ClientID)
	plan := upgradetypes.Plan{
		Name:   "ibc upgrade",
		Height: 1000,
	}

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {
				proposal, err = types.NewUpgradeProposal(ibctesting.Title, ibctesting.Description, plan, cs.ZeroCustomFields())
				suite.Require().NoError(err)
			}, true,
		},
		{
			"fails validate abstract - empty title", func() {
				proposal, err = types.NewUpgradeProposal("", ibctesting.Description, plan, cs.ZeroCustomFields())
				suite.Require().NoError(err)
			}, false,
		},
		{
			"non zeroed fields", func() {
				proposal, err = types.NewUpgradeProposal(ibctesting.Title, ibctesting.Description, plan, &ibctm.ClientState{
					FrozenHeight: types.Height{
						RevisionHeight: 10,
					},
				})
				suite.Require().NoError(err)
			}, false,
		},
		{
			"plan height is zero", func() {
				invalidPlan := upgradetypes.Plan{Name: "ibc upgrade", Height: 0}
				proposal, err = types.NewUpgradeProposal(ibctesting.Title, ibctesting.Description, invalidPlan, cs.ZeroCustomFields())
				suite.Require().NoError(err)
			}, false,
		},
		{
			"client state is nil", func() {
				proposal = &types.UpgradeProposal{
					Title:               ibctesting.Title,
					Description:         ibctesting.Description,
					Plan:                plan,
					UpgradedClientState: nil,
				}
			}, false,
		},
		{
			"failed to unpack client state", func() {
				any, err := types.PackConsensusState(&ibctm.ConsensusState{})
				suite.Require().NoError(err)

				proposal = &types.UpgradeProposal{
					Title:               ibctesting.Title,
					Description:         ibctesting.Description,
					Plan:                plan,
					UpgradedClientState: any,
				}
			}, false,
		},
	}

	for _, tc := range testCases {

		tc.malleate()

		err := proposal.ValidateBasic()

		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}

// tests an upgrade proposal can be marshaled and unmarshaled, and the
// client state can be unpacked
func (suite *TypesTestSuite) TestMarshalUpgradeProposal() {
	// create proposal
	plan := upgradetypes.Plan{
		Name:   "upgrade ibc",
		Height: 1000,
	}
	content, err := types.NewUpgradeProposal("title", "description", plan, &ibctm.ClientState{})
	suite.Require().NoError(err)

	up, ok := content.(*types.UpgradeProposal)
	suite.Require().True(ok)

	// create codec
	ir := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(ir)
	govtypes.RegisterInterfaces(ir)
	ibctm.RegisterInterfaces(ir)
	cdc := codec.NewProtoCodec(ir)

	// marshal message
	bz, err := cdc.MarshalJSON(up)
	suite.Require().NoError(err)

	// unmarshal proposal
	newUp := &types.UpgradeProposal{}
	err = cdc.UnmarshalJSON(bz, newUp)
	suite.Require().NoError(err)

	// unpack client state
	_, err = types.UnpackClientState(newUp.UpgradedClientState)
	suite.Require().NoError(err)
}

func (suite *TypesTestSuite) TestUpgradeString() {
	plan := upgradetypes.Plan{
		Name:   "ibc upgrade",
		Info:   "https://foo.bar/baz",
		Height: 1000,
	}

	proposal, err := types.NewUpgradeProposal(ibctesting.Title, ibctesting.Description, plan, &ibctm.ClientState{})
	suite.Require().NoError(err)

	expect := fmt.Sprintf("IBC Upgrade Proposal\n  Title: title\n  Description: description\n  Upgrade Plan\n  Name: ibc upgrade\n  height: 1000\n  Info: https://foo.bar/baz.\n  Upgraded IBC Client: %s", &ibctm.ClientState{})

	suite.Require().Equal(expect, proposal.String())
}
