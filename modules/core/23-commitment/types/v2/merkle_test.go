package v2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerklePathValidation(t *testing.T) {
	cases := []struct {
		name             string
		path             MerklePath
		expValidPrefix   bool
		expValidFullPath bool
	}{
		{
			"success: prefix and path",
			NewMerklePath([]byte("key1"), []byte("key2")),
			true,
			true,
		},
		{
			"success: prefix with empty last key",
			NewMerklePath([]byte("key1"), []byte("")),
			true,
			false,
		},
		{
			"success: prefix with single empty key",
			NewMerklePath([]byte("")),
			true,
			false,
		},
		{
			"failure: empty path",
			NewMerklePath(),
			false,
			false,
		},
		{
			"failure: empty key in start prefix",
			NewMerklePath([]byte(""), []byte("key2")),
			false,
			false,
		},
	}

	for _, tc := range cases {
		if tc.expValidPrefix {
			require.NoError(t, tc.path.ValidateAsPrefix(), tc.name)
		} else {
			require.Error(t, tc.path.ValidateAsPrefix(), tc.name)
		}

		if tc.expValidFullPath {
			require.NoError(t, tc.path.ValidateFullPath(), tc.name)
		} else {
			require.Error(t, tc.path.ValidateFullPath(), tc.name)
		}
	}
}
