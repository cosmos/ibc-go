package types_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

func (suite *TypesTestSuite) TestDenomsValidate() {
	testCases := []struct {
		name     string
		denoms   types.Denoms
		expError error
	}{
		{
			"empty Denoms",
			types.Denoms{},
			nil,
		},
		{
			"valid multiple trace info",
			types.Denoms{
				{Base: "uatom", Trace: []string{"transfer/channel-1", "transfer/channel-2"}},
			},
			nil,
		},
		{
			"valid multiple trace info",
			types.Denoms{
				{Base: "uatom", Trace: []string{"transfer/channel-1", "transfer/channel-2"}},
				{Base: "uatom", Trace: []string{"transfer/channel-1", "transfer/channel-2"}},
			},
			fmt.Errorf("duplicated denomination with hash"),
		},
		{
			"empty base denom with trace",
			types.Denoms{{Base: "", Trace: []string{"transfer/channel-1"}}},
			fmt.Errorf("base denomination cannot be blank"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			err := tc.denoms.Validate()
			if tc.expError == nil {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().ErrorContains(err, tc.expError.Error())
			}
		})
	}
}
