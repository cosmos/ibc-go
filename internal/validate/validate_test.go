package validate_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v9/internal/validate"
)

func TestGRPCRequest(t *testing.T) {
	const (
		validID   = "validIdentifier"
		invalidID = ""
	)
	testCases := []struct {
		msg       string
		portID    string
		channelID string
		expPass   bool
	}{
		{
			"success",
			validID,
			validID,
			true,
		},
		{
			"invalid portID",
			invalidID,
			validID,
			false,
		},
		{
			"invalid channelID",
			validID,
			invalidID,
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.msg), func(t *testing.T) {
			err := validate.GRPCRequest(tc.portID, tc.channelID)
			if tc.expPass {
				require.NoError(t, err, tc.msg)
			} else {
				require.Error(t, err, tc.msg)
			}
		})
	}
}
