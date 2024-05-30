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
				{Base: "uatom", Trace: []types.Trace{types.NewTrace("transfer", "channel-1"), types.NewTrace("transfer", "channel-2")}},
			},
			nil,
		},
		{
			"valid multiple trace info",
			types.Denoms{
				{Base: "uatom", Trace: []types.Trace{types.NewTrace("transfer", "channel-1"), types.NewTrace("transfer", "channel-2")}},
				{Base: "uatom", Trace: []types.Trace{types.NewTrace("transfer", "channel-1"), types.NewTrace("transfer", "channel-2")}},
			},
			fmt.Errorf("duplicated denomination with hash"),
		},
		{
			"empty base denom with trace",
			types.Denoms{{Base: "", Trace: []types.Trace{types.NewTrace("transfer", "channel-1")}}},
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

func (suite *TypesTestSuite) TestFullPath() {
	testCases := []struct {
		name    string
		denom   types.Denom
		expPath string
	}{
		{
			"empty Denom",
			types.Denom{},
			"",
		},
		{
			"only base denom",
			types.Denom{
				Base: "uatom",
			},
			"uatom",
		},
		{
			"base with slashes",
			types.Denom{
				Base: "gamm/pool/osmo",
			},
			"gamm/pool/osmo",
		},
		{
			"1 hop denom",
			types.Denom{
				Base:  "uatom",
				Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
			},
			"transfer/channel-0/uatom",
		},
		{
			"2 hop denom",
			types.Denom{
				Base:  "uatom",
				Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52")},
			},
			"transfer/channel-0/transfer/channel-52/uatom",
		},
		{
			"3 hop denom",
			types.Denom{
				Base:  "uatom",
				Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52"), types.NewTrace("transfer", "channel-52")},
			},
			"transfer/channel-0/transfer/channel-52/transfer/channel-52/uatom",
		},
		{
			"4 hop denom with base denom slashes",
			types.Denom{
				Base:  "other-denom/",
				Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52"), types.NewTrace("transfer", "channel-52"), types.NewTrace("transfer", "channel-49")},
			},
			"transfer/channel-0/transfer/channel-52/transfer/channel-52/transfer/channel-49/other-denom/",
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.Require().Equal(tc.expPath, tc.denom.FullPath())
		})
	}
}
