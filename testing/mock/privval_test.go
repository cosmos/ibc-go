package mock_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/ibc-go/v8/testing/mock"
)

const chainID = "testChain"

func TestGetPubKey(t *testing.T) {
	pv := mock.NewPV()
	pk, err := pv.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, "ed25519", pk.Type())
}

func TestSignVote(t *testing.T) {
	pv := mock.NewPV()
	pk, _ := pv.GetPubKey()

	vote := &cmtproto.Vote{Height: 2}
	err := pv.SignVote(chainID, vote)
	require.NoError(t, err)

	msg := cmttypes.VoteSignBytes(chainID, vote)
	ok := pk.VerifySignature(msg, vote.Signature)
	require.True(t, ok)
}

func TestSignProposal(t *testing.T) {
	pv := mock.NewPV()
	pk, _ := pv.GetPubKey()

	proposal := &cmtproto.Proposal{Round: 2}
	err := pv.SignProposal(chainID, proposal)
	require.NoError(t, err)

	msg := cmttypes.ProposalSignBytes(chainID, proposal)
	ok := pk.VerifySignature(msg, proposal.Signature)
	require.True(t, ok)
}
