package types_test

import (
	"fmt"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cometbft/cometbft/proto/tendermint/crypto"

	"github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
)

func (suite *MerkleTestSuite) TestConvertProofs() {
	suite.iavlStore.Set([]byte("MYKEY"), []byte("MYVALUE"))
	cid := suite.store.Commit()

	root := types.NewMerkleRoot(cid.Hash)
	existsPath := types.NewMerklePath([]byte(suite.storeKey.Name()), []byte("MYKEY"))
	nonexistPath := types.NewMerklePath([]byte(suite.storeKey.Name()), []byte("NOTMYKEY"))
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
				res, err := suite.store.Query(&storetypes.RequestQuery{
					Path:  fmt.Sprintf("/%s/key", suite.storeKey.Name()), // required path to get key/value+proof
					Data:  []byte("MYKEY"),
					Prove: true,
				})
				require.NoError(suite.T(), err)
				require.NotNil(suite.T(), res.ProofOps)

				proofOps = res.ProofOps
			},
			true, nil,
		},
		{
			"success for NonexistenceProof",
			func() {
				res, err := suite.store.Query(&storetypes.RequestQuery{
					Path:  fmt.Sprintf("/%s/key", suite.storeKey.Name()),
					Data:  []byte("NOTMYKEY"),
					Prove: true,
				})
				require.NoError(suite.T(), err)
				require.NotNil(suite.T(), res.ProofOps)

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
				res, err := suite.store.Query(&storetypes.RequestQuery{
					Path:  fmt.Sprintf("/%s/key", suite.storeKey.Name()), // required path to get key/value+proof
					Data:  []byte("MYKEY"),
					Prove: true,
				})
				require.NoError(suite.T(), err)
				require.NotNil(suite.T(), res.ProofOps)

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
			suite.Require().NoError(err, "ConvertProofs unexpectedly returned error for case: %s", tc.name)
			if tc.keyExists {
				err := proof.VerifyMembership(types.GetSDKSpecs(), &root, existsPath, value)
				suite.Require().NoError(err, "converted proof failed to verify membership for case: %s", tc.name)
			} else {
				err := proof.VerifyNonMembership(types.GetSDKSpecs(), &root, nonexistPath)
				suite.Require().NoError(err, "converted proof failed to verify non-membership for case: %s", tc.name)
			}
		} else {
			suite.Require().Error(err, "ConvertProofs passed on invalid case for case: %s", tc.name)
			suite.Require().ErrorIs(err, tc.expErr, "unexpected error returned for case: %s", tc.name)
		}
	}
}
