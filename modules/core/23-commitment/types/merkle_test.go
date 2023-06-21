package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
)

func (s *MerkleTestSuite) TestVerifyMembership() {
	s.iavlStore.Set([]byte("MYKEY"), []byte("MYVALUE"))
	cid := s.store.Commit()

	res := s.store.Query(abci.RequestQuery{
		Path:  fmt.Sprintf("/%s/key", s.storeKey.Name()), // required path to get key/value+proof
		Data:  []byte("MYKEY"),
		Prove: true,
	})
	require.NotNil(s.T(), res.ProofOps)

	proof, err := types.ConvertProofs(res.ProofOps)
	require.NoError(s.T(), err)

	s.Require().NoError(proof.ValidateBasic())
	s.Require().Error(types.MerkleProof{}.ValidateBasic())

	cases := []struct {
		name       string
		root       []byte
		pathArr    []string
		value      []byte
		malleate   func()
		shouldPass bool
	}{
		{"valid proof", cid.Hash, []string{s.storeKey.Name(), "MYKEY"}, []byte("MYVALUE"), func() {}, true},            // valid proof
		{"wrong value", cid.Hash, []string{s.storeKey.Name(), "MYKEY"}, []byte("WRONGVALUE"), func() {}, false},        // invalid proof with wrong value
		{"nil value", cid.Hash, []string{s.storeKey.Name(), "MYKEY"}, []byte(nil), func() {}, false},                   // invalid proof with nil value
		{"wrong key", cid.Hash, []string{s.storeKey.Name(), "NOTMYKEY"}, []byte("MYVALUE"), func() {}, false},          // invalid proof with wrong key
		{"wrong path 1", cid.Hash, []string{s.storeKey.Name(), "MYKEY", "MYKEY"}, []byte("MYVALUE"), func() {}, false}, // invalid proof with wrong path
		{"wrong path 2", cid.Hash, []string{s.storeKey.Name()}, []byte("MYVALUE"), func() {}, false},                   // invalid proof with wrong path
		{"wrong path 3", cid.Hash, []string{"MYKEY"}, []byte("MYVALUE"), func() {}, false},                             // invalid proof with wrong path
		{"wrong storekey", cid.Hash, []string{"otherStoreKey", "MYKEY"}, []byte("MYVALUE"), func() {}, false},          // invalid proof with wrong store prefix
		{"wrong root", []byte("WRONGROOT"), []string{s.storeKey.Name(), "MYKEY"}, []byte("MYVALUE"), func() {}, false}, // invalid proof with wrong root
		{"nil root", []byte(nil), []string{s.storeKey.Name(), "MYKEY"}, []byte("MYVALUE"), func() {}, false},           // invalid proof with nil root
		{"proof is wrong length", cid.Hash, []string{s.storeKey.Name(), "MYKEY"}, []byte("MYVALUE"), func() {
			proof = types.MerkleProof{
				Proofs: proof.Proofs[1:],
			}
		}, false}, // invalid proof with wrong length

	}

	for i, tc := range cases {
		tc := tc
		s.Run(tc.name, func() {
			tc.malleate()

			root := types.NewMerkleRoot(tc.root)
			path := types.NewMerklePath(tc.pathArr...)

			err := proof.VerifyMembership(types.GetSDKSpecs(), &root, path, tc.value)

			if tc.shouldPass {
				//nolint: scopelint
				s.Require().NoError(err, "test case %d should have passed", i)
			} else {
				//nolint: scopelint
				s.Require().Error(err, "test case %d should have failed", i)
			}
		})
	}
}

func (s *MerkleTestSuite) TestVerifyNonMembership() {
	s.iavlStore.Set([]byte("MYKEY"), []byte("MYVALUE"))
	cid := s.store.Commit()

	// Get Proof
	res := s.store.Query(abci.RequestQuery{
		Path:  fmt.Sprintf("/%s/key", s.storeKey.Name()), // required path to get key/value+proof
		Data:  []byte("MYABSENTKEY"),
		Prove: true,
	})
	require.NotNil(s.T(), res.ProofOps)

	proof, err := types.ConvertProofs(res.ProofOps)
	require.NoError(s.T(), err)

	s.Require().NoError(proof.ValidateBasic())

	cases := []struct {
		name       string
		root       []byte
		pathArr    []string
		malleate   func()
		shouldPass bool
	}{
		{"valid proof", cid.Hash, []string{s.storeKey.Name(), "MYABSENTKEY"}, func() {}, true},            // valid proof
		{"wrong key", cid.Hash, []string{s.storeKey.Name(), "MYKEY"}, func() {}, false},                   // invalid proof with existent key
		{"wrong path 1", cid.Hash, []string{s.storeKey.Name(), "MYKEY", "MYABSENTKEY"}, func() {}, false}, // invalid proof with wrong path
		{"wrong path 2", cid.Hash, []string{s.storeKey.Name(), "MYABSENTKEY", "MYKEY"}, func() {}, false}, // invalid proof with wrong path
		{"wrong path 3", cid.Hash, []string{s.storeKey.Name()}, func() {}, false},                         // invalid proof with wrong path
		{"wrong path 4", cid.Hash, []string{"MYABSENTKEY"}, func() {}, false},                             // invalid proof with wrong path
		{"wrong storeKey", cid.Hash, []string{"otherStoreKey", "MYABSENTKEY"}, func() {}, false},          // invalid proof with wrong store prefix
		{"wrong root", []byte("WRONGROOT"), []string{s.storeKey.Name(), "MYABSENTKEY"}, func() {}, false}, // invalid proof with wrong root
		{"nil root", []byte(nil), []string{s.storeKey.Name(), "MYABSENTKEY"}, func() {}, false},           // invalid proof with nil root
		{"proof is wrong length", cid.Hash, []string{s.storeKey.Name(), "MYKEY"}, func() {
			proof = types.MerkleProof{
				Proofs: proof.Proofs[1:],
			}
		}, false}, // invalid proof with wrong length

	}

	for i, tc := range cases {
		tc := tc

		s.Run(tc.name, func() {
			tc.malleate()

			root := types.NewMerkleRoot(tc.root)
			path := types.NewMerklePath(tc.pathArr...)

			err := proof.VerifyNonMembership(types.GetSDKSpecs(), &root, path)

			if tc.shouldPass {
				//nolint: scopelint
				s.Require().NoError(err, "test case %d should have passed", i)
			} else {
				//nolint: scopelint
				s.Require().Error(err, "test case %d should have failed", i)
			}
		})
	}
}

func TestApplyPrefix(t *testing.T) {
	prefix := types.NewMerklePrefix([]byte("storePrefixKey"))

	pathStr := "pathone/pathtwo/paththree/key"
	path := types.MerklePath{
		KeyPath: []string{pathStr},
	}

	prefixedPath, err := types.ApplyPrefix(prefix, path)
	require.NoError(t, err, "valid prefix returns error")

	require.Equal(t, "/storePrefixKey/"+pathStr, prefixedPath.Pretty(), "Prefixed path incorrect")
	require.Equal(t, "/storePrefixKey/pathone%2Fpathtwo%2Fpaththree%2Fkey", prefixedPath.String(), "Prefixed escaped path incorrect")
}

func TestString(t *testing.T) {
	path := types.NewMerklePath("rootKey", "storeKey", "path/to/leaf")

	require.Equal(t, "/rootKey/storeKey/path%2Fto%2Fleaf", path.String(), "path String returns unxpected value")
	require.Equal(t, "/rootKey/storeKey/path/to/leaf", path.Pretty(), "path's pretty string representation is incorrect")

	onePath := types.NewMerklePath("path/to/leaf")

	require.Equal(t, "/path%2Fto%2Fleaf", onePath.String(), "one element path does not have correct string representation")
	require.Equal(t, "/path/to/leaf", onePath.Pretty(), "one element path has incorrect pretty string representation")

	zeroPath := types.NewMerklePath()

	require.Equal(t, "", zeroPath.String(), "zero element path does not have correct string representation")
	require.Equal(t, "", zeroPath.Pretty(), "zero element path does not have correct pretty string representation")
}
