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
			types.Denoms{types.NewDenom("uatom", types.NewTrace("transfer", "channel-1"), types.NewTrace("transfer", "channel-2"))},
			nil,
		},
		{
			"valid multiple trace info",
			types.Denoms{
				types.NewDenom("uatom", types.NewTrace("transfer", "channel-1"), types.NewTrace("transfer", "channel-2")),
				types.NewDenom("uatom", types.NewTrace("transfer", "channel-1"), types.NewTrace("transfer", "channel-2")),
			},
			fmt.Errorf("duplicated denomination with hash"),
		},
		{
			"empty base denom with trace",
			types.Denoms{types.NewDenom("", types.NewTrace("transfer", "channel-1"))},
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

func (suite *TypesTestSuite) TestPath() {
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
			types.NewDenom("uatom"),
			"uatom",
		},
		{
			"base with slashes",
			types.NewDenom("gamm/pool/osmo"),
			"gamm/pool/osmo",
		},
		{
			"1 hop denom",
			types.NewDenom("uatom", types.NewTrace("transfer", "channel-0")),
			"transfer/channel-0/uatom",
		},
		{
			"2 hop denom",
			types.NewDenom("uatom", types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52")),
			"transfer/channel-0/transfer/channel-52/uatom",
		},
		{
			"3 hop denom",
			types.NewDenom("uatom", types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52"), types.NewTrace("transfer", "channel-52")),
			"transfer/channel-0/transfer/channel-52/transfer/channel-52/uatom",
		},
		{
			"4 hop denom with base denom slashes",
			types.NewDenom("other-denom/", types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52"), types.NewTrace("transfer", "channel-52"), types.NewTrace("transfer", "channel-49")),
			"transfer/channel-0/transfer/channel-52/transfer/channel-52/transfer/channel-49/other-denom/",
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.Require().Equal(tc.expPath, tc.denom.Path())
		})
	}
}

func (suite *TypesTestSuite) TestSort() {
	testCases := []struct {
		name      string
		denoms    types.Denoms
		expDenoms types.Denoms
	}{
		{
			"only base denom",
			types.Denoms{types.Denom{Base: "uosmo"}, types.Denom{Base: "gamm"}, types.Denom{Base: "uatom"}},
			types.Denoms{types.Denom{Base: "gamm"}, types.Denom{Base: "uatom"}, types.Denom{Base: "uosmo"}},
		},
		{
			"different base denom and same traces",
			types.Denoms{
				types.Denom{
					Base:  "uosmo",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
				},
				types.Denom{
					Base:  "gamm",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
				},
			},
			types.Denoms{
				types.Denom{
					Base:  "gamm",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
				},
				types.Denom{
					Base:  "uosmo",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
				},
			},
		},
		{
			"same base denom and different traces",
			types.Denoms{
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("mountain", "channel-0")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52"), types.NewTrace("transfer", "channel-52")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52")},
				},
				types.Denom{
					Base: "uatom",
				},
			},
			types.Denoms{
				types.Denom{
					Base: "uatom",
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("mountain", "channel-0")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52"), types.NewTrace("transfer", "channel-52")},
				},
			},
		},
		{
			"different base denoms and different traces",
			types.Denoms{
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
				},
				types.Denom{
					Base:  "gamm",
					Trace: []types.Trace{types.NewTrace("pool", "channel-0")},
				},
				types.Denom{
					Base:  "gamm",
					Trace: []types.Trace{types.NewTrace("pool", "channel-0"), types.NewTrace("transfer", "channel-52")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52"), types.NewTrace("transfer", "channel-52")},
				},
				types.Denom{
					Base: "utia",
				},
				types.Denom{
					Base:  "gamm",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52")},
				},
			},
			types.Denoms{
				types.Denom{
					Base:  "gamm",
					Trace: []types.Trace{types.NewTrace("pool", "channel-0")},
				},
				types.Denom{
					Base:  "gamm",
					Trace: []types.Trace{types.NewTrace("pool", "channel-0"), types.NewTrace("transfer", "channel-52")},
				},
				types.Denom{
					Base:  "gamm",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0")},
				},
				types.Denom{
					Base:  "uatom",
					Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52"), types.NewTrace("transfer", "channel-52")},
				},
				types.Denom{
					Base: "utia",
				},
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.Require().Equal(tc.expDenoms, tc.denoms.Sort())
		})
	}
}

func (suite *TypesTestSuite) TestDenomChainSource() {
	testCases := []struct {
		name                     string
		denom                    types.Denom
		sourcePort               string
		sourceChannel            string
		expReceiverChainIsSource bool
	}{
		{
			"sender chain is source: empty trace",
			types.NewDenom("uatom", []types.Trace{}...),
			"transfer",
			"channel-0",
			false,
		},
		{
			"sender chain is source: nil trace",
			types.NewDenom("uatom"),
			"transfer",
			"channel-0",
			false,
		},
		{
			"sender chain is source: single trace",
			types.NewDenom("ubtc", types.NewTrace("transfer", "channel-1")),
			"transfer",
			"channel-0",
			false,
		},
		{
			"sender chain is source: swapped portID and channelID",
			types.NewDenom("uatom", types.NewTrace("transfer", "channel-0")),
			"channel-0",
			"transfer",
			false,
		},
		{
			"sender chain is source: multi-trace",
			types.NewDenom("uatom", types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52")),
			"transfer",
			"channel-1",
			false,
		},
		{
			"receiver chain is source: single trace",
			types.NewDenom(
				"factory/stars16da2uus9zrsy83h23ur42v3lglg5rmyrpqnju4/dust",
				types.NewTrace("transfer", "channel-0"),
			),
			"transfer",
			"channel-0",
			true,
		},
		{
			"receiver chain is source: multi-trace",
			types.NewDenom("uatom", types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-52")),
			"transfer",
			"channel-0",
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.Require().Equal(tc.expReceiverChainIsSource, tc.denom.ReceiverChainIsSource(tc.sourcePort, tc.sourceChannel))
			suite.Require().Equal(!tc.expReceiverChainIsSource, tc.denom.SenderChainIsSource(tc.sourcePort, tc.sourceChannel))
		})
	}
}
