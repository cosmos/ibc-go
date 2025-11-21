package types_test

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"

	"github.com/cometbft/cometbft/proto/tendermint/crypto"

	"github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
)

func (s *MerkleTestSuite) TestConvertProofs() {
	s.iavlStore.Set([]byte("MYKEY"), []byte("MYVALUE"))
	cid := s.store.Commit()

	root := types.NewMerkleRoot(cid.Hash)
	existsPath := types.NewMerklePath([]byte(s.storeKey.Name()), []byte("MYKEY"))
	nonexistPath := types.NewMerklePath([]byte(s.storeKey.Name()), []byte("NOTMYKEY"))
	value := []byte("MYVALUE")

	var proofOps *crypto.ProofOps
	testcases := []struct {
		name      string
		malleate  func()
		keyExists bool
		expErr    error
	}{
		{
			"success for ExistenceProof",
			func() {
				res, err := s.store.Query(&storetypes.RequestQuery{
					Path:  fmt.Sprintf("/%s/key", s.storeKey.Name()), // required path to get key/value+proof
					Data:  []byte("MYKEY"),
					Prove: true,
				})
				s.Require().NoError(err)
				s.Require().NotNil(res.ProofOps)

				proofOps = res.ProofOps
			},
			true, nil,
		},
		{
			"success for NonexistenceProof",
			func() {
				res, err := s.store.Query(&storetypes.RequestQuery{
					Path:  fmt.Sprintf("/%s/key", s.storeKey.Name()),
					Data:  []byte("NOTMYKEY"),
					Prove: true,
				})
				s.Require().NoError(err)
				s.Require().NotNil(res.ProofOps)

				proofOps = res.ProofOps
			},
			false, nil,
		},
		{
			"nil proofOps",
			func() {
				proofOps = nil
			},
			true, types.ErrInvalidMerkleProof,
		},
		{
			"proof op data is nil",
			func() {
				res, err := s.store.Query(&storetypes.RequestQuery{
					Path:  fmt.Sprintf("/%s/key", s.storeKey.Name()), // required path to get key/value+proof
					Data:  []byte("MYKEY"),
					Prove: true,
				})
				s.Require().NoError(err)
				s.Require().NotNil(s.T(), res.ProofOps)

				proofOps = res.ProofOps
				proofOps.Ops[0].Data = nil
			},
			true, types.ErrInvalidMerkleProof,
		},
	}

	for _, tc := range testcases {
		tc.malleate()

		proof, err := types.ConvertProofs(proofOps)
		if tc.expErr == nil {
			s.Require().NoError(err, "ConvertProofs unexpectedly returned error for case: %s", tc.name)
			if tc.keyExists {
				err := proof.VerifyMembership(types.GetSDKSpecs(), &root, existsPath, value)
				s.Require().NoError(err, "converted proof failed to verify membership for case: %s", tc.name)
			} else {
				err := proof.VerifyNonMembership(types.GetSDKSpecs(), &root, nonexistPath)
				s.Require().NoError(err, "converted proof failed to verify non-membership for case: %s", tc.name)
			}
		} else {
			s.Require().Error(err, "ConvertProofs passed on invalid case for case: %s", tc.name)
			s.Require().ErrorIs(err, tc.expErr, "unexpected error returned for case: %s", tc.name)
		}
	}
}
