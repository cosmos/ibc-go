package ibctesting

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmprotostate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

// ApplyValSetChanges takes in tmtypes.ValidatorSet and []abci.ValidatorUpdate and will return a new tmtypes.ValidatorSet which has the
// provided validator updates applied to the provided validator set.
func ApplyValSetChanges(t *testing.T, valSet *tmtypes.ValidatorSet, valUpdates []abci.ValidatorUpdate) *tmtypes.ValidatorSet {
	updates, err := tmtypes.PB2TM.ValidatorUpdates(valUpdates)
	require.NoError(t, err)

	// must copy since validator set will mutate with UpdateWithChangeSet
	newVals := valSet.Copy()
	err = newVals.UpdateWithChangeSet(updates)
	require.NoError(t, err)

	return newVals
}

// ABCIResponsesResultsHash returns a merkle hash of ABCI results
func ABCIResponsesResultsHash(ar *tmprotostate.ABCIResponses) []byte {
	return tmtypes.NewResults(ar.DeliverTxs).Hash()
}

// MakeCommit iterates over the provided validator set, creating a Precommit vote for each
// participant at the provided height and round. Each vote is signed and added to the VoteSet.
// Finally, the VoteSet is committed finalizing the block.
func MakeCommit(ctx context.Context, blockID tmtypes.BlockID, height int64, round int32, voteSet *tmtypes.VoteSet, validators []tmtypes.PrivValidator, now time.Time) (*tmtypes.Commit, error) {
	// all sign
	for i := 0; i < len(validators); i++ {
		pubKey, err := validators[i].GetPubKey()
		if err != nil {
			return nil, err
		}
		vote := &tmtypes.Vote{
			ValidatorAddress: pubKey.Address(),
			ValidatorIndex:   int32(i),
			Height:           height,
			Round:            round,
			Type:             tmproto.PrecommitType,
			BlockID:          blockID,
			Timestamp:        now,
		}

		v := vote.ToProto()

		if err := validators[i].SignVote(ctx, voteSet.ChainID(), v); err != nil {
			return nil, err
		}
		vote.Signature = v.Signature
		if _, err := voteSet.AddVote(vote); err != nil {
			return nil, err
		}
	}

	return voteSet.MakeCommit(), nil
}
