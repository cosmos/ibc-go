package tendermint_test

import (
	"time"

	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

func (s *TendermintTestSuite) TestConsensusStateValidateBasic() {
	testCases := []struct {
		msg            string
		consensusState *ibctm.ConsensusState
		expectPass     bool
	}{
		{
			"success",
			&ibctm.ConsensusState{
				Timestamp:          s.now,
				Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
				NextValidatorsHash: s.valsHash,
			},
			true,
		},
		{
			"success with sentinel",
			&ibctm.ConsensusState{
				Timestamp:          s.now,
				Root:               commitmenttypes.NewMerkleRoot([]byte(ibctm.SentinelRoot)),
				NextValidatorsHash: s.valsHash,
			},
			true,
		},
		{
			"root is nil",
			&ibctm.ConsensusState{
				Timestamp:          s.now,
				Root:               commitmenttypes.MerkleRoot{},
				NextValidatorsHash: s.valsHash,
			},
			false,
		},
		{
			"root is empty",
			&ibctm.ConsensusState{
				Timestamp:          s.now,
				Root:               commitmenttypes.MerkleRoot{},
				NextValidatorsHash: s.valsHash,
			},
			false,
		},
		{
			"nextvalshash is invalid",
			&ibctm.ConsensusState{
				Timestamp:          s.now,
				Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
				NextValidatorsHash: []byte("hi"),
			},
			false,
		},

		{
			"timestamp is zero",
			&ibctm.ConsensusState{
				Timestamp:          time.Time{},
				Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
				NextValidatorsHash: s.valsHash,
			},
			false,
		},
	}

	for i, tc := range testCases {
		s.Run(tc.msg, func() {
			// check just to increase coverage
			s.Require().Equal(exported.Tendermint, tc.consensusState.ClientType())
			s.Require().Equal(tc.consensusState.GetRoot(), tc.consensusState.Root)

			err := tc.consensusState.ValidateBasic()
			if tc.expectPass {
				s.Require().NoError(err, "valid test case %d failed: %s", i, tc.msg)
			} else {
				s.Require().Error(err, "invalid test case %d passed: %s", i, tc.msg)
			}
		})
	}
}
