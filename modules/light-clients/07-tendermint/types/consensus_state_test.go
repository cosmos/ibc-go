package types_test

import (
	"time"

<<<<<<< HEAD:modules/light-clients/07-tendermint/types/consensus_state_test.go
	commitmenttypes "github.com/cosmos/ibc-go/v5/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v5/modules/core/exported"
	"github.com/cosmos/ibc-go/v5/modules/light-clients/07-tendermint/types"
=======
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint"
>>>>>>> c86d27f (chore: increment go mod to v6 (#2318)):modules/light-clients/07-tendermint/consensus_state_test.go
)

func (suite *TendermintTestSuite) TestConsensusStateValidateBasic() {
	testCases := []struct {
		msg            string
		consensusState *types.ConsensusState
		expectPass     bool
	}{
		{
			"success",
			&types.ConsensusState{
				Timestamp:          suite.now,
				Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
				NextValidatorsHash: suite.valsHash,
			},
			true,
		},
		{
			"success with sentinel",
			&types.ConsensusState{
				Timestamp:          suite.now,
				Root:               commitmenttypes.NewMerkleRoot([]byte(types.SentinelRoot)),
				NextValidatorsHash: suite.valsHash,
			},
			true,
		},
		{
			"root is nil",
			&types.ConsensusState{
				Timestamp:          suite.now,
				Root:               commitmenttypes.MerkleRoot{},
				NextValidatorsHash: suite.valsHash,
			},
			false,
		},
		{
			"root is empty",
			&types.ConsensusState{
				Timestamp:          suite.now,
				Root:               commitmenttypes.MerkleRoot{},
				NextValidatorsHash: suite.valsHash,
			},
			false,
		},
		{
			"nextvalshash is invalid",
			&types.ConsensusState{
				Timestamp:          suite.now,
				Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
				NextValidatorsHash: []byte("hi"),
			},
			false,
		},

		{
			"timestamp is zero",
			&types.ConsensusState{
				Timestamp:          time.Time{},
				Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
				NextValidatorsHash: suite.valsHash,
			},
			false,
		},
	}

	for i, tc := range testCases {
		tc := tc

		// check just to increase coverage
		suite.Require().Equal(exported.Tendermint, tc.consensusState.ClientType())
		suite.Require().Equal(tc.consensusState.GetRoot(), tc.consensusState.Root)

		err := tc.consensusState.ValidateBasic()
		if tc.expectPass {
			suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.msg)
		} else {
			suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.msg)
		}
	}
}
