package celestia

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

var _ exported.ClientState = (*ClientState)(nil)

// ClientType implements exported.ClientState.
func (*ClientState) ClientType() string {
	return ModuleName
}

// Validate implements exported.ClientState.
func (cs *ClientState) Validate() error {
	return cs.BaseClient.Validate()
}

// VerifyMembership is a generic proof verification method which verifies an NMT proof
// that a set of shares exist in a set of rows and a Merkle proof that those rows exist
// in a Merkle tree with a given data root.
// TODO: Revise and look into delay periods for this.
// TODO: Validate key path and value against the shareProof extracted from proof bytes.
func (cs *ClientState) VerifyMembership(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, height exported.Height, delayTimePeriod uint64, delayBlockPeriod uint64, proof []byte, path exported.Path, value []byte) error {
	if cs.BaseClient.LatestHeight.LT(height) {
		return errorsmod.Wrapf(
			ibcerrors.ErrInvalidHeight,
			"client state height < proof height (%d < %d), please ensure the client has been updated", cs.BaseClient.LatestHeight, height,
		)
	}

	if err := verifyDelayPeriodPassed(ctx, clientStore, height, delayTimePeriod, delayBlockPeriod); err != nil {
		return err
	}

	var shareProofProto ShareProof
	if err := cdc.Unmarshal(proof, &shareProofProto); err != nil {
		return errorsmod.Wrapf(commitmenttypes.ErrInvalidProof, "could not unmarshal share proof: %v", err)
	}

	shareProof, err := ShareProofFromProto(&shareProofProto)
	if err != nil {
		return err
	}

	consensusState, found := ibctm.GetConsensusState(clientStore, cdc, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "please ensure the proof was constructed against a height that exists on the client")
	}

	return shareProof.Validate(consensusState.GetRoot().GetHash())
}

// verifyDelayPeriodPassed will ensure that at least delayTimePeriod amount of time and delayBlockPeriod number of blocks have passed
// since consensus state was submitted before allowing verification to continue.
func verifyDelayPeriodPassed(ctx sdk.Context, store storetypes.KVStore, proofHeight exported.Height, delayTimePeriod, delayBlockPeriod uint64) error {
	if delayTimePeriod != 0 {
		// check that executing chain's timestamp has passed consensusState's processed time + delay time period
		processedTime, ok := ibctm.GetProcessedTime(store, proofHeight)
		if !ok {
			return errorsmod.Wrapf(ibctm.ErrProcessedTimeNotFound, "processed time not found for height: %s", proofHeight)
		}

		currentTimestamp := uint64(ctx.BlockTime().UnixNano())
		validTime := processedTime + delayTimePeriod

		// NOTE: delay time period is inclusive, so if currentTimestamp is validTime, then we return no error
		if currentTimestamp < validTime {
			return errorsmod.Wrapf(ibctm.ErrDelayPeriodNotPassed, "cannot verify packet until time: %d, current time: %d",
				validTime, currentTimestamp)
		}
	}

	if delayBlockPeriod != 0 {
		// check that executing chain's height has passed consensusState's processed height + delay block period
		processedHeight, ok := ibctm.GetProcessedHeight(store, proofHeight)
		if !ok {
			return errorsmod.Wrapf(ibctm.ErrProcessedHeightNotFound, "processed height not found for height: %s", proofHeight)
		}

		currentHeight := clienttypes.GetSelfHeight(ctx)
		validHeight := clienttypes.NewHeight(processedHeight.GetRevisionNumber(), processedHeight.GetRevisionHeight()+delayBlockPeriod)

		// NOTE: delay block period is inclusive, so if currentHeight is validHeight, then we return no error
		if currentHeight.LT(validHeight) {
			return errorsmod.Wrapf(ibctm.ErrDelayPeriodNotPassed, "cannot verify packet until height: %s, current height: %s",
				validHeight, currentHeight)
		}
	}

	return nil
}
