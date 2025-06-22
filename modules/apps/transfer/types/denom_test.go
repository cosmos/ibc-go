package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

func (s *TypesTestSuite) TestDenomsValidate() {
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
			"valid trace with client id",
			types.Denoms{types.NewDenom("uatom", types.NewHop("transfer", "07-tendermint-0"))},
			nil,
		},
		{
			"valid multiple trace info",
			types.Denoms{types.NewDenom("uatom", types.NewHop("transfer", "channel-1"), types.NewHop("transfer", "channel-2"))},
			nil,
		},
		{
			"valid multiple trace info",
			types.Denoms{
				types.NewDenom("uatom", types.NewHop("transfer", "channel-1"), types.NewHop("transfer", "channel-2")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-1"), types.NewHop("transfer", "channel-2")),
			},
			errors.New("duplicated denomination with hash"),
		},
		{
			"empty base denom with trace",
			types.Denoms{types.NewDenom("", types.NewHop("transfer", "channel-1"))},
			errors.New("base denomination cannot be blank"),
		},
	}

	for _, tc := range testCases {
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

func (s *TypesTestSuite) TestPath() {
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
			types.NewDenom("uatom", types.NewHop("transfer", "channel-0")),
			"transfer/channel-0/uatom",
		},
		{
			"1 hop denom with client id",
			types.NewDenom("uatom", types.NewHop("transfer", "07-tendermint-0")),
			"transfer/07-tendermint-0/uatom",
		},
		{
			"1 hop denom with client id and slashes",
			types.NewDenom("gamm/pool/osmo", types.NewHop("transfer", "07-tendermint-0")),
			"transfer/07-tendermint-0/gamm/pool/osmo",
		},
		{
			"2 hop denom",
			types.NewDenom("uatom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52")),
			"transfer/channel-0/transfer/channel-52/uatom",
		},
		{
			"3 hop denom",
			types.NewDenom("uatom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52"), types.NewHop("transfer", "channel-52")),
			"transfer/channel-0/transfer/channel-52/transfer/channel-52/uatom",
		},
		{
			"4 hop denom with base denom slashes",
			types.NewDenom("other-denom/", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52"), types.NewHop("transfer", "channel-52"), types.NewHop("transfer", "channel-49")),
			"transfer/channel-0/transfer/channel-52/transfer/channel-52/transfer/channel-49/other-denom/",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.expPath, tc.denom.Path())
		})
	}
}

func (s *TypesTestSuite) TestSort() {
	testCases := []struct {
		name      string
		denoms    types.Denoms
		expDenoms types.Denoms
	}{
		{
			"only base denom",
			types.Denoms{types.NewDenom("uosmo"), types.NewDenom("gamm"), types.NewDenom("uatom")},
			types.Denoms{types.NewDenom("gamm"), types.NewDenom("uatom"), types.NewDenom("uosmo")},
		},
		{
			"different base denom and same traces",
			types.Denoms{
				types.NewDenom("uosmo", types.NewHop("transfer", "channel-0")),
				types.NewDenom("gamm", types.NewHop("transfer", "channel-0")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0")),
			},
			types.Denoms{
				types.NewDenom("gamm", types.NewHop("transfer", "channel-0")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0")),
				types.NewDenom("uosmo", types.NewHop("transfer", "channel-0")),
			},
		},
		{
			"same base denom and different traces",
			types.Denoms{
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0")),
				types.NewDenom("uatom", types.NewHop("mountain", "channel-0")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52"), types.NewHop("transfer", "channel-52")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52")),
				types.NewDenom("uatom"),
			},
			types.Denoms{
				types.NewDenom("uatom"),
				types.NewDenom("uatom", types.NewHop("mountain", "channel-0")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52"), types.NewHop("transfer", "channel-52")),
			},
		},
		{
			"different base denoms and different traces",
			types.Denoms{
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0")),
				types.NewDenom("gamm", types.NewHop("pool", "channel-0")),
				types.NewDenom("gamm", types.NewHop("pool", "channel-0"), types.NewHop("transfer", "channel-52")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52"), types.NewHop("transfer", "channel-52")),
				types.NewDenom("utia"),
				types.NewDenom("gamm", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52")),
			},
			types.Denoms{
				types.NewDenom("gamm", types.NewHop("pool", "channel-0")),
				types.NewDenom("gamm", types.NewHop("pool", "channel-0"), types.NewHop("transfer", "channel-52")),
				types.NewDenom("gamm", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0")),
				types.NewDenom("uatom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52"), types.NewHop("transfer", "channel-52")),
				types.NewDenom("utia"),
			},
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.expDenoms, tc.denoms.Sort())
		})
	}
}

func (s *TypesTestSuite) TestDenomChainSource() {
	testCases := []struct {
		name          string
		denom         types.Denom
		sourcePort    string
		sourceChannel string
		expHasPrefix  bool
	}{
		{
			"sender chain is source: empty trace",
			types.NewDenom("uatom", []types.Hop{}...),
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
			types.NewDenom("ubtc", types.NewHop("transfer", "channel-1")),
			"transfer",
			"channel-0",
			false,
		},
		{
			"sender chain is source: single trace with client id",
			types.NewDenom("ubtc", types.NewHop("transfer", "07-tendermint-0")),
			"transfer",
			"channel-0",
			false,
		},
		{
			"sender chain is source: swapped portID and channelID",
			types.NewDenom("uatom", types.NewHop("transfer", "channel-0")),
			"channel-0",
			"transfer",
			false,
		},
		{
			"sender chain is source: multi-trace",
			types.NewDenom("uatom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52")),
			"transfer",
			"channel-1",
			false,
		},
		{
			"receiver chain is source: single trace",
			types.NewDenom(
				"factory/stars16da2uus9zrsy83h23ur42v3lglg5rmyrpqnju4/dust",
				types.NewHop("transfer", "channel-0"),
			),
			"transfer",
			"channel-0",
			true,
		},
		{
			"receiver chain is source: single trace with client id",
			types.NewDenom("ubtc", types.NewHop("transfer", "07-tendermint-0")),
			"transfer",
			"07-tendermint-0",
			true,
		},
		{
			"receiver chain is source: multi-trace",
			types.NewDenom("uatom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-52")),
			"transfer",
			"channel-0",
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.expHasPrefix, tc.denom.HasPrefix(tc.sourcePort, tc.sourceChannel))
		})
	}
}

func TestValidateIBCDenom(t *testing.T) {
	testCases := []struct {
		name     string
		denom    string
		expError bool
	}{
		{"denom with trace hash", "ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", false},
		{"base denom", "uatom", false},
		{"base denom ending with '/'", "uatom/", false},
		{"base denom with single '/'s", "gamm/pool/1", false},
		{"base denom with double '/'s", "gamm//pool//1", false},
		{"non-ibc prefix with hash", "notibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", false},
		{"empty denom", "", true},
		{"denom 'ibc'", "ibc", true},
		{"denom 'ibc/'", "ibc/", true},
		{"invalid hash", "ibc/!@#$!@#", true},
	}

	for _, tc := range testCases {
		err := types.ValidateIBCDenom(tc.denom)
		if tc.expError {
			require.Error(t, err, tc.name)
			continue
		}
		require.NoError(t, err, tc.name)
	}
}

func TestExtractDenomFromPath(t *testing.T) {
	testCases := []struct {
		name     string
		fullPath string
		expDenom types.Denom
	}{
		{"empty denom", "", types.Denom{}},
		{"base denom no slashes", "atom", types.NewDenom("atom")},
		{"base denom with trailing slash", "atom/", types.NewDenom("atom/")},
		{"base denom multiple trailing slash", "foo///bar//baz/atom/", types.NewDenom("foo///bar//baz/atom/")},
		{"ibc denom one hop", "transfer/channel-0/atom", types.NewDenom("atom", types.NewHop("transfer", "channel-0"))},
		{"ibc denom one hop with client id", "transfer/07-tendermint-0/atom", types.NewDenom("atom", types.NewHop("transfer", "07-tendermint-0"))},
		{"ibc denom one hop trailing slash", "transfer/channel-0/atom/", types.NewDenom("atom/", types.NewHop("transfer", "channel-0"))},
		{"ibc denom one hop multiple slashes", "transfer/channel-0//at/om/", types.NewDenom("/at/om/", types.NewHop("transfer", "channel-0"))},
		{"ibc denom two hops", "transfer/channel-0/transfer/channel-60/atom", types.NewDenom("atom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-60"))},
		{"ibc denom two hops trailing slash", "transfer/channel-0/transfer/channel-60/atom/", types.NewDenom("atom/", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-60"))},
		{"empty prefix", "/uatom", types.NewDenom("/uatom")},
		{"empty identifiers", "//uatom", types.NewDenom("//uatom")},
		{"base denom with single '/'", "erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", types.NewDenom("erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA")},
		{"trace info and base denom with single '/'", "transfer/channel-1/erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", types.NewDenom("erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", types.NewHop("transfer", "channel-1"))},
		{"single trace identifier", "transfer/", types.NewDenom("transfer/")},
		{"trace info with custom port", "customtransfer/channel-1/uatom", types.NewDenom("uatom", types.NewHop("customtransfer", "channel-1"))},
		{"invalid path (1)", "channel-1/transfer/uatom", types.NewDenom("channel-1/transfer/uatom")},
		{"invalid path (2)", "transfer/channel-1", types.NewDenom("transfer/channel-1")},
		{"invalid path (3)", "transfer/channel-1/transfer/channel-2", types.NewDenom("", types.NewHop("transfer", "channel-1"), types.NewHop("transfer", "channel-2"))},
		{"invalid path (4)", "transfer/channelToA/uatom", types.NewDenom("transfer/channelToA/uatom")},
	}

	for _, tc := range testCases {
		denom := types.ExtractDenomFromPath(tc.fullPath)
		require.Equal(t, tc.expDenom, denom, tc.name)
	}
}
