package types_test

import (
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

func (suite *WasmTestSuite) TestConsensusStateValidateBasic() {
	testCases := []struct {
		msg            string
		consensusState *types.ConsensusState
		expectPass     bool
	}{
		{
			"success",
			&types.ConsensusState{
				Timestamp: uint64(suite.now.Unix()),
				Root:      &commitmenttypes.MerkleRoot{Hash: []byte("app_hash")},
				Data:      []byte("data"),
				CodeId:    []byte("codeid"),
			},
			true,
		},
		{
			"root is nil",
			&types.ConsensusState{
				Timestamp: uint64(suite.now.Unix()),
				Root:      nil,
				Data:      []byte("data"),
				CodeId:    []byte("codeid"),
			},
			false,
		},
		{
			"root is empty",
			&types.ConsensusState{
				Timestamp: uint64(suite.now.Unix()),
				Root:      &commitmenttypes.MerkleRoot{},
				Data:      []byte("data"),
				CodeId:    []byte("codeid"),
			},
			false,
		},
		{
			"timestamp is zero",
			&types.ConsensusState{
				Timestamp: 0,
				Root:      &commitmenttypes.MerkleRoot{Hash: []byte("app_hash")},
				Data:      []byte("data"),
				CodeId:    []byte("codeid"),
			},
			false,
		},
		{
			"data is nil",
			&types.ConsensusState{
				Timestamp: uint64(suite.now.Unix()),
				Root:      &commitmenttypes.MerkleRoot{Hash: []byte("app_hash")},
				Data:      nil,
				CodeId:    []byte("codeid"),
			},
			false,
		},
		{
			"data is empty",
			&types.ConsensusState{
				Timestamp: uint64(suite.now.Unix()),
				Root:      &commitmenttypes.MerkleRoot{Hash: []byte("app_hash")},
				Data:      []byte(""),
				CodeId:    []byte("codeid"),
			},
			false,
		},
		{
			"code id is nil",
			&types.ConsensusState{
				Timestamp: uint64(suite.now.Unix()),
				Root:      &commitmenttypes.MerkleRoot{Hash: []byte("app_hash")},
				Data:      []byte("data"),
				CodeId:    nil,
			},
			false,
		},
		{
			"code id is empty",
			&types.ConsensusState{
				Timestamp: uint64(suite.now.Unix()),
				Root:      &commitmenttypes.MerkleRoot{Hash: []byte("app_hash")},
				Data:      []byte("data"),
				CodeId:    []byte(""),
			},
			false,
		},
	}

	for i, tc := range testCases {
		tc := tc

		// check just to increase coverage
		suite.Require().Equal(exported.Wasm, tc.consensusState.ClientType())
		suite.Require().Equal(tc.consensusState.GetRoot(), tc.consensusState.Root)

		err := tc.consensusState.ValidateBasic()
		if tc.expectPass {
			suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.msg)
		} else {
			suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.msg)
		}
	}
}
