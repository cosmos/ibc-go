package v3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePrefixedDenom(t *testing.T) {
	testCases := []struct {
		name     string
		token    Token
		expError bool
	}{
		{
			"base denom",
			Token{
				Denom:  "atom",
				Amount: "1000",
				Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
			},
			false,
		},
		// {"prefixed denom", "transfer/channel-1/uatom", false},
		// {"prefixed denom with '/'", "transfer/channel-1/gamm/pool/1", false},
		// {"empty prefix", "/uatom", false},
		// {"empty identifiers", "//uatom", false},
		// {"base denom", "uatom", false},
		// {"base denom with single '/'", "erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", false},
		// {"base denom with multiple '/'s", "gamm/pool/1", false},
		// {"invalid port ID", "(transfer)/channel-1/uatom", true},
		// {"empty denom", "", true},
		// {"single trace identifier", "transfer/", true},
	}

	for _, tc := range testCases {
		tc := tc

		err := ValidateToken(tc.token)
		if tc.expError {
			require.Error(t, err, tc.name)
			continue
		}
		require.NoError(t, err, tc.name)
	}
}
