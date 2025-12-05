package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
)

func (s *MerkleTestSuite) TestVerifyMembership() {
	s.iavlStore.Set([]byte("MYKEY"), []byte("MYVALUE"))
	cid := s.store.Commit()

	res, err := s.store.Query(&storetypes.RequestQuery{
		Path:  fmt.Sprintf("/%s/key", s.storeKey.Name()), // required path to get key/value+proof
		Data:  []byte("MYKEY"),
		Prove: true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(res.ProofOps)

	proof, err := types.ConvertProofs(res.ProofOps)
	s.Require().NoError(err)

	cases := []struct {
		name       string
		root       []byte
		pathArr    [][]byte
		value      []byte
		malleate   func()
		shouldPass bool
	}{
		{"valid proof", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("MYKEY")}, []byte("MYVALUE"), func() {}, true},                    // valid proof
		{"wrong value", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("MYKEY")}, []byte("WRONGVALUE"), func() {}, false},                // invalid proof with wrong value
		{"nil value", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("MYKEY")}, []byte(nil), func() {}, false},                           // invalid proof with nil value
		{"wrong key", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("NOTMYKEY")}, []byte("MYVALUE"), func() {}, false},                  // invalid proof with wrong key
		{"wrong path 1", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("MYKEY"), []byte("MYKEY")}, []byte("MYVALUE"), func() {}, false}, // invalid proof with wrong path
		{"wrong path 2", cid.Hash, [][]byte{[]byte(s.storeKey.Name())}, []byte("MYVALUE"), func() {}, false},                                   // invalid proof with wrong path
		{"wrong path 3", cid.Hash, [][]byte{[]byte("MYKEY")}, []byte("MYVALUE"), func() {}, false},                                             // invalid proof with wrong path
		{"wrong storekey", cid.Hash, [][]byte{[]byte("otherStoreKey"), []byte("MYKEY")}, []byte("MYVALUE"), func() {}, false},                  // invalid proof with wrong store prefix
		{"wrong root", []byte("WRONGROOT"), [][]byte{[]byte(s.storeKey.Name()), []byte("MYKEY")}, []byte("MYVALUE"), func() {}, false},         // invalid proof with wrong root
		{"nil root", []byte(nil), [][]byte{[]byte(s.storeKey.Name()), []byte("MYKEY")}, []byte("MYVALUE"), func() {}, false},                   // invalid proof with nil root
		{"proof is wrong length", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("MYKEY")}, []byte("MYVALUE"), func() {
			proof = types.MerkleProof{
				Proofs: proof.Proofs[1:],
			}
		}, false}, // invalid proof with wrong length

	}

	for i, tc := range cases {
		s.Run(tc.name, func() {
			tc.malleate()

			root := types.NewMerkleRoot(tc.root)
			path := types.NewMerklePath(tc.pathArr...)

			err := proof.VerifyMembership(types.GetSDKSpecs(), &root, path, tc.value)

			if tc.shouldPass {
				// nolint: scopelint
				s.Require().NoError(err, "test case %d should have passed", i)
			} else {
				// nolint: scopelint
				s.Require().Error(err, "test case %d should have failed", i)
			}
		})
	}
}

func (s *MerkleTestSuite) TestVerifyNonMembership() {
	s.iavlStore.Set([]byte("MYKEY"), []byte("MYVALUE"))
	cid := s.store.Commit()

	// Get Proof
	res, err := s.store.Query(&storetypes.RequestQuery{
		Path:  fmt.Sprintf("/%s/key", s.storeKey.Name()), // required path to get key/value+proof
		Data:  []byte("MYABSENTKEY"),
		Prove: true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(res.ProofOps)

	proof, err := types.ConvertProofs(res.ProofOps)
	s.Require().NoError(err)

	cases := []struct {
		name       string
		root       []byte
		pathArr    [][]byte
		malleate   func()
		shouldPass bool
	}{
		{"valid proof", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("MYABSENTKEY")}, func() {}, true},                    // valid proof
		{"wrong key", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("MYKEY")}, func() {}, false},                           // invalid proof with existent key
		{"wrong path 1", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("MYKEY"), []byte("MYABSENTKEY")}, func() {}, false}, // invalid proof with wrong path
		{"wrong path 2", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("MYABSENTKEY"), []byte("MYKEY")}, func() {}, false}, // invalid proof with wrong path
		{"wrong path 3", cid.Hash, [][]byte{[]byte(s.storeKey.Name())}, func() {}, false},                                         // invalid proof with wrong path
		{"wrong path 4", cid.Hash, [][]byte{[]byte("MYABSENTKEY")}, func() {}, false},                                             // invalid proof with wrong path
		{"wrong storeKey", cid.Hash, [][]byte{[]byte("otherStoreKey"), []byte("MYABSENTKEY")}, func() {}, false},                  // invalid proof with wrong store prefix
		{"wrong root", []byte("WRONGROOT"), [][]byte{[]byte(s.storeKey.Name()), []byte("MYABSENTKEY")}, func() {}, false},         // invalid proof with wrong root
		{"nil root", []byte(nil), [][]byte{[]byte(s.storeKey.Name()), []byte("MYABSENTKEY")}, func() {}, false},                   // invalid proof with nil root
		{"proof is wrong length", cid.Hash, [][]byte{[]byte(s.storeKey.Name()), []byte("MYKEY")}, func() {
			proof = types.MerkleProof{
				Proofs: proof.Proofs[1:],
			}
		}, false}, // invalid proof with wrong length

	}

	for i, tc := range cases {
		s.Run(tc.name, func() {
			tc.malleate()

			root := types.NewMerkleRoot(tc.root)
			path := types.NewMerklePath(tc.pathArr...)

			err := proof.VerifyNonMembership(types.GetSDKSpecs(), &root, path)

			if tc.shouldPass {
				// nolint: scopelint
				s.Require().NoError(err, "test case %d should have passed", i)
			} else {
				// nolint: scopelint
				s.Require().Error(err, "test case %d should have failed", i)
			}
		})
	}
}

func TestApplyPrefix(t *testing.T) {
	prefix := types.NewMerklePrefix([]byte("storePrefixKey"))

	pathBz := []byte("pathone/pathtwo/paththree/key")
	path := v2.MerklePath{
		KeyPath: [][]byte{pathBz},
	}

	prefixedPath, err := types.ApplyPrefix(prefix, path)
	require.NoError(t, err, "valid prefix returns error")
	require.Len(t, prefixedPath.GetKeyPath(), 2, "unexpected key path length")

	key0, err := prefixedPath.GetKey(0)
	require.NoError(t, err, "get key 0 returns error")
	require.Equal(t, prefix.KeyPrefix, key0, "key 0 does not match expected value")

	key1, err := prefixedPath.GetKey(1)
	require.NoError(t, err, "get key 1 returns error")
	require.Equal(t, pathBz, key1, "key 1 does not match expected value")
}
