package types_test

import (
	fmt "fmt"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

func (s *TypesTestSuite) Denoms_Validate() {
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
			fmt.Errorf("duplicated denomination trace with hash"),
		},
		{
			"empty base denom with trace",
			types.Denoms{{Base: "", Trace: []string{"transfer/channel-1"}}},
			fmt.Errorf("base denomination cannot be blank"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			err := tc.denoms.Validate()
			if tc.expError == nil {
				s.Require().NoError(err, tc.name)
			} else {
				s.Require().ErrorContains(err, tc.expError.Error())
			}
		})
	}
}
