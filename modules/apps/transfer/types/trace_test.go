package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDenomTrace(t *testing.T) {
	testCases := []struct {
		name     string
		denom    string
		expTrace DenomTrace
	}{
		{"empty denom", "", DenomTrace{}},
		{"base denom", "uatom", DenomTrace{BaseDenom: "uatom"}},
		{"base denom ending with '/'", "uatom/", DenomTrace{BaseDenom: "uatom/"}},
		{"base denom with single '/'s", "gamm/pool/1", DenomTrace{BaseDenom: "gamm/pool/1"}},
		{"base denom with double '/'s", "gamm//pool//1", DenomTrace{BaseDenom: "gamm//pool//1"}},
		{"trace info", "transfer/channel-0/uatom", DenomTrace{BaseDenom: "uatom", Path: "transfer/channel-0"}},
		{"trace info with base denom ending in '/'", "transfer/channel-0/uatom/", DenomTrace{BaseDenom: "uatom/", Path: "transfer/channel-0"}},
		{"trace info with single '/' in base denom", "transfer/channel-0/erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", DenomTrace{BaseDenom: "erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", Path: "transfer/channel-0"}},
		{"trace info with multiple '/'s in base denom", "transfer/channel-0/gamm/pool/1", DenomTrace{BaseDenom: "gamm/pool/1", Path: "transfer/channel-0"}},
		{"trace info with multiple double '/'s in base denom", "transfer/channel-0/gamm//pool//1", DenomTrace{BaseDenom: "gamm//pool//1", Path: "transfer/channel-0"}},
		{"trace info with multiple port/channel pairs", "transfer/channel-0/transfer/channel-1/uatom", DenomTrace{BaseDenom: "uatom", Path: "transfer/channel-0/transfer/channel-1"}},
		{"incomplete path", "transfer/uatom", DenomTrace{BaseDenom: "transfer/uatom"}},
		{"invalid path (1)", "transfer//uatom", DenomTrace{BaseDenom: "transfer//uatom"}},
		{"invalid path (2)", "transfer/channelToA/uatom", DenomTrace{BaseDenom: "transfer/channelToA/uatom"}},
		{"invalid path (3)", "channel-0/transfer/uatom", DenomTrace{BaseDenom: "channel-0/transfer/uatom"}},
		{"invalid path (4)", "uatom/transfer", DenomTrace{BaseDenom: "uatom/transfer"}},
		{"invalid path (5)", "transfer/channel-0", DenomTrace{Path: "transfer/channel-0"}},
		{"invalid path (6)", "transfer/channel-0/", DenomTrace{Path: "transfer/channel-0"}},
		{"invalid path (7)", "transfer/channelToA", DenomTrace{BaseDenom: "transfer/channelToA"}},
		{"invalid path (8)", "transfer/channel-0/transfer", DenomTrace{BaseDenom: "transfer", Path: "transfer/channel-0"}},
		{"invalid path (9)", "transfer/channel-0/transfer/channel-1", DenomTrace{Path: "transfer/channel-0/transfer/channel-1"}},
		{"invalid path (10)", "transfer/channel-0/transfer/channelToB", DenomTrace{BaseDenom: "transfer/channelToB", Path: "transfer/channel-0"}},
	}

	for _, tc := range testCases {
		trace := ParseDenomTrace(tc.denom)
		require.Equal(t, tc.expTrace, trace, tc.name)
	}
}

func TestDenomTrace_IBCDenom(t *testing.T) {
	testCases := []struct {
		name     string
		trace    DenomTrace
		expDenom string
	}{
		{"base denom", DenomTrace{BaseDenom: "uatom"}, "uatom"},
		{"trace info", DenomTrace{BaseDenom: "uatom", Path: "transfer/channelToA"}, "ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2"},
	}

	for _, tc := range testCases {
		denom := tc.trace.IBCDenom()
		require.Equal(t, tc.expDenom, denom, tc.name)
	}
}

func TestDenomTrace_Validate(t *testing.T) {
	testCases := []struct {
		name     string
		trace    DenomTrace
		expError bool
	}{
		{"base denom only", DenomTrace{BaseDenom: "uatom"}, false},
		{"empty DenomTrace", DenomTrace{}, true},
		{"valid single trace info", DenomTrace{BaseDenom: "uatom", Path: "transfer/channelToA"}, false},
		{"valid multiple trace info", DenomTrace{BaseDenom: "uatom", Path: "transfer/channelToA/transfer/channelToB"}, false},
		{"single trace identifier", DenomTrace{BaseDenom: "uatom", Path: "transfer"}, true},
		{"invalid port ID", DenomTrace{BaseDenom: "uatom", Path: "(transfer)/channelToA"}, true},
		{"invalid channel ID", DenomTrace{BaseDenom: "uatom", Path: "transfer/(channelToA)"}, true},
		{"empty base denom with trace", DenomTrace{BaseDenom: "", Path: "transfer/channelToA"}, true},
	}

	for _, tc := range testCases {
		err := tc.trace.Validate()
		if tc.expError {
			require.Error(t, err, tc.name)
			continue
		}
		require.NoError(t, err, tc.name)
	}
}

func TestTraces_Validate(t *testing.T) {
	testCases := []struct {
		name     string
		traces   Traces
		expError bool
	}{
		{"empty Traces", Traces{}, false},
		{"valid multiple trace info", Traces{{BaseDenom: "uatom", Path: "transfer/channelToA/transfer/channelToB"}}, false},
		{
			"valid multiple trace info",
			Traces{
				{BaseDenom: "uatom", Path: "transfer/channelToA/transfer/channelToB"},
				{BaseDenom: "uatom", Path: "transfer/channelToA/transfer/channelToB"},
			},
			true,
		},
		{"empty base denom with trace", Traces{{BaseDenom: "", Path: "transfer/channelToA"}}, true},
	}

	for _, tc := range testCases {
		err := tc.traces.Validate()
		if tc.expError {
			require.Error(t, err, tc.name)
			continue
		}
		require.NoError(t, err, tc.name)
	}
}

func TestValidatePrefixedDenom(t *testing.T) {
	testCases := []struct {
		name     string
		denom    string
		expError bool
	}{
		{"prefixed denom", "transfer/channelToA/uatom", false},
		{"base denom", "uatom", false},
		{"empty denom", "", true},
		{"empty prefix", "/uatom", true},
		{"empty identifiers", "//uatom", true},
		{"single trace identifier", "transfer/", true},
		{"invalid port ID", "(transfer)/channelToA/uatom", true},
		{"invalid channel ID", "transfer/(channelToA)/uatom", true},
	}

	for _, tc := range testCases {
		err := ValidatePrefixedDenom(tc.denom)
		if tc.expError {
			require.Error(t, err, tc.name)
			continue
		}
		require.NoError(t, err, tc.name)
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
		{"invald hash", "ibc/!@#$!@#", true},
	}

	for _, tc := range testCases {
		err := ValidateIBCDenom(tc.denom)
		if tc.expError {
			require.Error(t, err, tc.name)
			continue
		}
		require.NoError(t, err, tc.name)
	}
}
