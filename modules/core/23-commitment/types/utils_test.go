package types_test

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	crypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
)

func (s *MerkleTestSuite) TestConvertProofs() {
	s.iavlStore.Set([]byte("MYKEY"), []byte("MYVALUE"))
	cid := s.store.Commit()

	root := types.NewMerkleRoot(cid.Hash)
	existsPath := types.NewMerklePath(s.storeKey.Name(), "MYKEY")
	nonexistPath := types.NewMerklePath(s.storeKey.Name(), "NOTMYKEY")
	value := []byte("MYVALUE")

	var proofOps *crypto.ProofOps
	testcases := []struct {
		name      string
		malleate  func()
		keyExists bool
		expPass   bool
	}{
		{
			"success for ExistenceProof",
			func() {
				res := s.store.Query(abci.RequestQuery{
					Path:  fmt.Sprintf("/%s/key", s.storeKey.Name()), // required path to get key/value+proof
					Data:  []byte("MYKEY"),
					Prove: true,
				})
				require.NotNil(s.T(), res.ProofOps)

				proofOps = res.ProofOps
			},
			true, true,
		},
		{
			"success for NonexistenceProof",
			func() {
				res := s.store.Query(abci.RequestQuery{
					Path:  fmt.Sprintf("/%s/key", s.storeKey.Name()), // required path to get key/value+proof
					Data:  []byte("NOTMYKEY"),
					Prove: true,
				})
				require.NotNil(s.T(), res.ProofOps)

				proofOps = res.ProofOps
			},
			false, true,
		},
		{
			"nil proofOps",
			func() {
				proofOps = nil
			},
			true, false,
		},
		{
			"proof op data is nil",
			func() {
				res := s.store.Query(abci.RequestQuery{
					Path:  fmt.Sprintf("/%s/key", s.storeKey.Name()), // required path to get key/value+proof
					Data:  []byte("MYKEY"),
					Prove: true,
				})
				require.NotNil(s.T(), res.ProofOps)

				proofOps = res.ProofOps
				proofOps.Ops[0].Data = nil
			},
			true, false,
		},
	}

	for _, tc := range testcases {
		tc.malleate()

		proof, err := types.ConvertProofs(proofOps)
		if tc.expPass {
			s.Require().NoError(err, "ConvertProofs unexpectedly returned error for case: %s", tc.name)
			if tc.keyExists {
				err := proof.VerifyMembership(types.GetSDKSpecs(), &root, existsPath, value)
				s.Require().NoError(err, "converted proof failed to verify membership for case: %s", tc.name)
			} else {
				err := proof.VerifyNonMembership(types.GetSDKSpecs(), &root, nonexistPath)
				s.Require().NoError(err, "converted proof failed to verify membership for case: %s", tc.name)
			}
		} else {
			s.Require().Error(err, "ConvertProofs passed on invalid case for case: %s", tc.name)
		}
	}
}
