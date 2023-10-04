package ibctesting

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	abci "github.com/cometbft/cometbft/abci/types"
	tmtypes "github.com/cometbft/cometbft/types"
)

// ApplyValSetChanges takes in tmtypes.ValidatorSet and []abci.ValidatorUpdate and will return a new tmtypes.ValidatorSet which has the
// provided validator updates applied to the provided validator set.
func ApplyValSetChanges(tb testing.TB, valSet *tmtypes.ValidatorSet, valUpdates []abci.ValidatorUpdate) *tmtypes.ValidatorSet {
	tb.Helper()
	updates, err := tmtypes.PB2TM.ValidatorUpdates(valUpdates)
	require.NoError(tb, err)

	// must copy since validator set will mutate with UpdateWithChangeSet
	newVals := valSet.Copy()
	err = newVals.UpdateWithChangeSet(updates)
	require.NoError(tb, err)

	return newVals
}

// VoteAndCheckProposalStatus votes on a gov proposal, checks if the proposal has passed, and returns an error if it has not with the failure reason.
func VoteAndCheckProposalStatus(endpoint *Endpoint, proposalID uint64) error {
	// vote on proposal
	ctx := endpoint.Chain.GetContext()
	require.NoError(endpoint.Chain.TB, endpoint.Chain.GetSimApp().GovKeeper.AddVote(ctx, proposalID, endpoint.Chain.SenderAccount.GetAddress(), govtypesv1.NewNonSplitVoteOption(govtypesv1.OptionYes), ""))

	// fast forward the chain context to end the voting period
	params, err := endpoint.Chain.GetSimApp().GovKeeper.Params.Get(ctx)
	require.NoError(endpoint.Chain.TB, err)

	endpoint.Chain.Coordinator.IncrementTimeBy(*params.VotingPeriod + *params.MaxDepositPeriod)
	newHeader := endpoint.Chain.GetContext().BlockHeader()
	ctx = ctx.WithBlockHeader(newHeader)
	endpoint.Chain.NextBlock()

	// check if proposal passed or failed on msg execution
	p, err := endpoint.Chain.GetSimApp().GovKeeper.Proposals.Get(ctx, proposalID)
	require.NoError(endpoint.Chain.TB, err)
	if p.Status != govtypesv1.StatusPassed {
		return fmt.Errorf("proposal failed: %s", p.FailedReason)
	}
	return nil
}
