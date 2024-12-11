package validate_test

import (
	"errors"
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
		expErr    error
	}{
		{
			"success",
			validID,
			validID,
			nil,
		},
		{
			"invalid portID",
			invalidID,
			validID,
			errors.New("rpc error: code = InvalidArgument desc = identifier cannot be blank: invalid identifier"),
		},
		{
			"invalid channelID",
			validID,
			invalidID,
			errors.New("rpc error: code = InvalidArgument desc = identifier cannot be blank: invalid identifier"),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.msg), func(t *testing.T) {
			err := validate.GRPCRequest(tc.portID, tc.channelID)
			if tc.expErr == nil {
				require.NoError(t, err, tc.msg)
			} else {
				require.Error(t, err, tc.msg)
				require.Equal(t, err.Error(), tc.expErr.Error())
			}
		})
	}
}
