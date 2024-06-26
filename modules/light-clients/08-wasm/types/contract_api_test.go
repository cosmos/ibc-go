package types_test

import (
	"encoding/json"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
)

func (suite *TypesTestSuite) TestVerifyMembershipMsgEncoding() {
	msg := types.VerifyMembershipMsg{
		Height: clienttypes.NewHeight(1, 100),
		Proof:  []byte("proofbytes"),
		Path:   commitmenttypes.MerklePath{KeyPath: []string{"ibc", "/key/path"}},
		Value:  []byte("value"),
	}

	bz, err := json.Marshal(msg)
	suite.Require().NoError(err)

	expected := `{"height":{"revision_number":1,"revision_height":100},"delay_time_period":0,"delay_block_period":0,"proof":"cHJvb2ZieXRlcw==","path":{"key_path":["ibc","/key/path"]},"value":"dmFsdWU="}`
	suite.Require().Equal(expected, string(bz))

	merklePath := commitmenttypes.NewMerklePath([]byte("ibc"), []byte("/key/path"))
	msg = types.VerifyMembershipMsg{
		Height:     clienttypes.NewHeight(1, 100),
		Proof:      []byte("proofbytes"),
		MerklePath: &merklePath,
		Value:      []byte("value"),
	}

	bz, err = json.Marshal(msg)
	suite.Require().NoError(err)

	expected = `{"height":{"revision_number":1,"revision_height":100},"delay_time_period":0,"delay_block_period":0,"proof":"cHJvb2ZieXRlcw==","merkle_path":{"key_path":["aWJj","L2tleS9wYXRo"]},"path":{},"value":"dmFsdWU="}`
	suite.Require().Equal(expected, string(bz))
}

func (suite *TypesTestSuite) TestVerifyNonMembershipMsgEncoding() {
	msg := types.VerifyNonMembershipMsg{
		Height: clienttypes.NewHeight(1, 100),
		Proof:  []byte("proofbytes"),
		Path:   commitmenttypes.MerklePath{KeyPath: []string{"ibc", "/key/path"}},
	}

	bz, err := json.Marshal(msg)
	suite.Require().NoError(err)

	expected := `{"height":{"revision_number":1,"revision_height":100},"delay_time_period":0,"delay_block_period":0,"proof":"cHJvb2ZieXRlcw==","path":{"key_path":["ibc","/key/path"]}}`
	suite.Require().Equal(expected, string(bz))

	merklePath := commitmenttypes.NewMerklePath([]byte("ibc"), []byte("/key/path"))
	msg = types.VerifyNonMembershipMsg{
		Height:     clienttypes.NewHeight(1, 100),
		Proof:      []byte("proofbytes"),
		MerklePath: &merklePath,
	}

	bz, err = json.Marshal(msg)
	suite.Require().NoError(err)

	expected = `{"height":{"revision_number":1,"revision_height":100},"delay_time_period":0,"delay_block_period":0,"proof":"cHJvb2ZieXRlcw==","merkle_path":{"key_path":["aWJj","L2tleS9wYXRo"]},"path":{}}`
	suite.Require().Equal(expected, string(bz))
}
