package validate_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/ibc-go/v10/internal/validate"
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
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"invalid channelID",
			validID,
			invalidID,
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.msg), func(t *testing.T) {
			err := validate.GRPCRequest(tc.portID, tc.channelID)

			if tc.expErr == nil {
				require.NoError(t, err, tc.msg)
			} else {
				require.Error(t, err, tc.msg)
				require.EqualError(t, err, tc.expErr.Error(), tc.msg)
			}
		})
	}
}
