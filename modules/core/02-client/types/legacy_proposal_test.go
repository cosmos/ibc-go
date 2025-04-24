package types_test

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (suite *TypesTestSuite) TestValidateBasic() {
	subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
	subjectPath.SetupClients()
	subject := subjectPath.EndpointA.ClientID

	substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
	substitutePath.SetupClients()
	substitute := substitutePath.EndpointA.ClientID

	testCases := []struct {
		name     string
		proposal govv1beta1.Content
		expErr   error
	}{
		{
			"success",
			types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, substitute),
			nil,
		},
		{
			"fails validate abstract - empty title",
			types.NewClientUpdateProposal("", ibctesting.Description, subject, substitute),
			govtypes.ErrInvalidProposalContent,
		},
		{
			"subject and substitute use the same identifier",
			types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, subject),
			types.ErrInvalidSubstitute,
		},
		{
			"invalid subject clientID",
			types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, ibctesting.InvalidID, substitute),
			host.ErrInvalidID,
		},
		{
			"invalid substitute clientID",
			types.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subject, ibctesting.InvalidID),
			host.ErrInvalidID,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.proposal.ValidateBasic()

			if tc.expErr == nil {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().ErrorIs(err, tc.expErr, tc.name)
			}
		})
	}
}

// tests a client update proposal can be marshaled and unmarshaled
func (suite *TypesTestSuite) TestMarshalClientUpdateProposalProposal() {
	// create proposal
	proposal := types.NewClientUpdateProposal("update IBC client", "description", "subject", "substitute")

	// create codec
	ir := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(ir)
	govv1beta1.RegisterInterfaces(ir)
	cdc := codec.NewProtoCodec(ir)

	// marshal message
	content, ok := proposal.(*types.ClientUpdateProposal)
	suite.Require().True(ok)
	bz, err := cdc.MarshalJSON(content)
	suite.Require().NoError(err)

	// unmarshal proposal
	newProposal := &types.ClientUpdateProposal{}
	err = cdc.UnmarshalJSON(bz, newProposal)
	suite.Require().NoError(err)
}
