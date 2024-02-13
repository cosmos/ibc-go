package keeper

import (
	"reflect"

	errorsmod "cosmossdk.io/errors"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/light"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

var _ types.SelfClientValidator = (*TendermintClientValidator)(nil)

// TendermintClientValidator implements the SelfClientValidator interface.
type TendermintClientValidator struct {
	stakingKeeper types.StakingKeeper
}

// NewTendermintClientValidator creates and returns a new SelfClientValidator for tendermint consensus.
func NewTendermintClientValidator(stakingKeeper types.StakingKeeper) *TendermintClientValidator {
	return &TendermintClientValidator{
		stakingKeeper: stakingKeeper,
	}
}

// GetSelfConsensusState implements types.SelfClientValidatorI.
func (tcv *TendermintClientValidator) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
	selfHeight, ok := height.(types.Height)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", types.Height{}, height)
	}

	// check that height revision matches chainID revision
	revision := types.ParseChainID(ctx.ChainID())
	if revision != height.GetRevisionNumber() {
		return nil, errorsmod.Wrapf(types.ErrInvalidHeight, "chainID revision number does not match height revision number: expected %d, got %d", revision, height.GetRevisionNumber())
	}

	histInfo, err := tcv.stakingKeeper.GetHistoricalInfo(ctx, int64(selfHeight.RevisionHeight))
	if err != nil {
		return nil, errorsmod.Wrapf(err, "height %d", selfHeight.RevisionHeight)
	}

	consensusState := &ibctm.ConsensusState{
		Timestamp:          histInfo.Header.Time,
		Root:               commitmenttypes.NewMerkleRoot(histInfo.Header.GetAppHash()),
		NextValidatorsHash: histInfo.Header.NextValidatorsHash,
	}

	return consensusState, nil
}

// ValidateSelfClient implements types.SelfClientValidatorI.
func (tcv *TendermintClientValidator) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	tmClient, ok := clientState.(*ibctm.ClientState)
	if !ok {
		return errorsmod.Wrapf(types.ErrInvalidClient, "client must be a Tendermint client, expected: %T, got: %T", &ibctm.ClientState{}, tmClient)
	}

	if !tmClient.FrozenHeight.IsZero() {
		return types.ErrClientFrozen
	}

	if ctx.ChainID() != tmClient.ChainId {
		return errorsmod.Wrapf(types.ErrInvalidClient, "invalid chain-id. expected: %s, got: %s",
			ctx.ChainID(), tmClient.ChainId)
	}

	revision := types.ParseChainID(ctx.ChainID())

	// client must be in the same revision as executing chain
	if tmClient.LatestHeight.RevisionNumber != revision {
		return errorsmod.Wrapf(types.ErrInvalidClient, "client is not in the same revision as the chain. expected revision: %d, got: %d",
			tmClient.LatestHeight.RevisionNumber, revision)
	}

	selfHeight := types.NewHeight(revision, uint64(ctx.BlockHeight()))
	if tmClient.LatestHeight.GTE(selfHeight) {
		return errorsmod.Wrapf(types.ErrInvalidClient, "client has LatestHeight %d greater than or equal to chain height %d",
			tmClient.LatestHeight, selfHeight)
	}

	expectedProofSpecs := commitmenttypes.GetSDKSpecs()
	if !reflect.DeepEqual(expectedProofSpecs, tmClient.ProofSpecs) {
		return errorsmod.Wrapf(types.ErrInvalidClient, "client has invalid proof specs. expected: %v got: %v",
			expectedProofSpecs, tmClient.ProofSpecs)
	}

	if err := light.ValidateTrustLevel(tmClient.TrustLevel.ToTendermint()); err != nil {
		return errorsmod.Wrapf(types.ErrInvalidClient, "trust-level invalid: %v", err)
	}

	expectedUbdPeriod, err := tcv.stakingKeeper.UnbondingTime(ctx)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to retrieve unbonding period")
	}

	if expectedUbdPeriod != tmClient.UnbondingPeriod {
		return errorsmod.Wrapf(types.ErrInvalidClient, "invalid unbonding period. expected: %s, got: %s",
			expectedUbdPeriod, tmClient.UnbondingPeriod)
	}

	if tmClient.UnbondingPeriod < tmClient.TrustingPeriod {
		return errorsmod.Wrapf(types.ErrInvalidClient, "unbonding period must be greater than trusting period. unbonding period (%d) < trusting period (%d)",
			tmClient.UnbondingPeriod, tmClient.TrustingPeriod)
	}

	if len(tmClient.UpgradePath) != 0 {
		// For now, SDK IBC implementation assumes that upgrade path (if defined) is defined by SDK upgrade module
		expectedUpgradePath := []string{upgradetypes.StoreKey, upgradetypes.KeyUpgradedIBCState}
		if !reflect.DeepEqual(expectedUpgradePath, tmClient.UpgradePath) {
			return errorsmod.Wrapf(types.ErrInvalidClient, "upgrade path must be the upgrade path defined by upgrade module. expected %v, got %v",
				expectedUpgradePath, tmClient.UpgradePath)
		}
	}

	return nil
}
