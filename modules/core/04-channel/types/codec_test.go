package types_test

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	ibcmock "github.com/cosmos/ibc-go/v3/testing/mock"
)

type caseAny struct {
	name    string
	any     *codectypes.Any
	expPass bool
}

func (suite *TypesTestSuite) TestPackAcknowledgement() {

	testCases := []struct {
		name            string
		acknowledgement exported.Acknowledgement
		expPass         bool
	}{
		{
			"success",
			&ibcmock.MockAcknowledgement,
			true,
		},
		{
			"nil",
			nil,
			false,
		},
	}

	testCasesAny := []caseAny{}

	for _, tc := range testCases {
		ackAny, err := types.PackAcknowledgement(tc.acknowledgement)
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}

		testCasesAny = append(testCasesAny, caseAny{tc.name, ackAny, tc.expPass})
	}

	for i, tc := range testCasesAny {
		cs, err := types.UnpackAcknowledgement(tc.any)
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
			suite.Require().Equal(testCases[i].acknowledgement, cs, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}
