package validate_test

import (
	"testing"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/require"
)

func TestGRPCRequest(t* testing.T) {
	const (
		validID = "validIdentifier"
		invalidID = ""
	)
	testCases := []struct {
		msg string
		portID string
		channelID string
		expError error
	} {
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
			status.Error(codes.InvalidArgument),
		},
	}


	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.msg), func(t * testing.T) {
			err := GRPCRequest(tc.portID, tc.channelID)
			if tc.expError == nil {
				require.NoError(t, err, tc.msg)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
	}
}