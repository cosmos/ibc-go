package v2_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
)

func TestMerklePathValidation(t *testing.T) {
	cases := []struct {
		name         string
		path         commitmenttypesv2.MerklePath
		expPrefixErr error
		expPathErr   error
	}{
		{
			"success: prefix and path",
			commitmenttypesv2.NewMerklePath([]byte("key1"), []byte("key2")),
			nil,
			nil,
		},
		{
			"prefix with empty last key",
			commitmenttypesv2.NewMerklePath([]byte("key1"), []byte("")),
			nil,
			errors.New("key at index 1 cannot be empty"),
		},
		{
			"prefix with single empty key",
			commitmenttypesv2.NewMerklePath([]byte("")),
			nil,
			errors.New("key at index 0 cannot be empty"),
		},
		{
			"failure: empty path",
			commitmenttypesv2.NewMerklePath(),
			errors.New("path cannot have length 0"),
			errors.New("path cannot have length 0"),
		},
		{
			"failure: prefix with empty first key",
			commitmenttypesv2.NewMerklePath([]byte(""), []byte("key2")),
			errors.New("key at index 0 cannot be empty"),
			errors.New("key at index 0 cannot be empty"),
		},
	}

	for _, tc := range cases {
		err := tc.path.ValidateAsPrefix()
		if tc.expPrefixErr == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorContains(t, err, tc.expPrefixErr.Error(), tc.name)
		}

		err = tc.path.ValidateAsPath()
		if tc.expPathErr == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorContains(t, err, tc.expPathErr.Error(), tc.name)
		}
	}
}
