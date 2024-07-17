package tendermint

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ clienttypes.ConsensusHost = (*ConsensusHost)(nil)

// ConsensusHost implements the 02-client clienttypes.ConsensusHost interface.
type ConsensusHost struct {
	stakingKeeper StakingKeeper
}

// StakingKeeper defines an expected interface for the tendermint ConsensusHost.
type StakingKeeper interface {
	GetHistoricalInfo(ctx context.Context, height int64) (stakingtypes.HistoricalInfo, error)
	UnbondingTime(ctx context.Context) (time.Duration, error)
}

// NewConsensusHost creates and returns a new ConsensusHost for tendermint consensus.
func NewConsensusHost(stakingKeeper clienttypes.StakingKeeper) clienttypes.ConsensusHost {
	if stakingKeeper == nil {
		panic("staking keeper cannot be nil")
	}

	return &ConsensusHost{
		stakingKeeper: stakingKeeper,
	}
}

// GetSelfConsensusState implements the 02-client clienttypes.ConsensusHost interface.
func (c *ConsensusHost) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
	selfHeight, ok := height.(clienttypes.Height)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", clienttypes.Height{}, height)
	}

	// check that height revision matches chainID revision
	revision := clienttypes.ParseChainID(ctx.ChainID())
	if revision != height.GetRevisionNumber() {
		return nil, errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "chainID revision number does not match height revision number: expected %d, got %d", revision, height.GetRevisionNumber())
	}

	histInfo, err := c.stakingKeeper.GetHistoricalInfo(ctx, int64(selfHeight.RevisionHeight))
	if err != nil {
		return nil, errorsmod.Wrapf(err, "height %d", selfHeight.RevisionHeight)
	}

	consensusState := &ConsensusState{
		Timestamp:          histInfo.Header.Time,
		Root:               commitmenttypes.NewMerkleRoot(histInfo.Header.GetAppHash()),
		NextValidatorsHash: histInfo.Header.NextValidatorsHash,
	}

	return consensusState, nil
}
