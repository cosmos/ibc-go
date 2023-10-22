package avalanche

import (
	"bytes"
	"math/big"
	"reflect"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// CheckForMisbehaviour detects duplicate height misbehaviour and BFT time violation misbehaviour
// in a submitted Header message and verifies the correctness of a submitted Misbehaviour ClientMessage
func (cs ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, msg exported.ClientMessage) bool {
	switch msg := msg.(type) {
	case *Header:
		avaHeader := msg
		consState := avaHeader.ConsensusState()

		uniqPrevVdrs, _, err := ValidateValidatorSet(ctx, avaHeader.PrevSubnetHeader.PchainVdrs)
		if err != nil {
			return true
		}
		uniqVdrs, _, err := ValidateValidatorSet(ctx, avaHeader.SubnetHeader.PchainVdrs)
		if err != nil {
			return true
		}

		// TODO check 2/3 vdrs 1 msg or all msg
		numberTrustedVdrs := 0
		for i := range uniqPrevVdrs {
			for m := range uniqVdrs {
				if reflect.DeepEqual(uniqPrevVdrs[i].PublicKeyBytes, uniqVdrs[m].PublicKeyBytes) {
					numberTrustedVdrs = numberTrustedVdrs + 1
				}
			}
		}

		scaledNumberTrustedVdrs := new(big.Int).SetInt64(int64(numberTrustedVdrs))
		scaledNumberTrustedVdrs.Mul(scaledNumberTrustedVdrs, new(big.Int).SetUint64(3))
		scaledVdrsLen := new(big.Int).SetUint64(uint64(len(uniqVdrs)))
		scaledVdrsLen.Mul(scaledVdrsLen, new(big.Int).SetUint64(2))

		if scaledNumberTrustedVdrs.Cmp(scaledVdrsLen) != 1 {
			return true
		}

		// Check if the Client store already has a consensus state for the header's height
		// If the consensus state exists, and it matches the header then we return early
		// since header has already been submitted in a previous UpdateClient.
		existingConsState, _ := GetConsensusState(clientStore, cdc, avaHeader.SubnetHeader.Height)
		if existingConsState != nil {
			// This header has already been submitted and the necessary state is already stored
			// in client store, thus we can return early without further validation.
			if reflect.DeepEqual(existingConsState, avaHeader.ConsensusState()) { //nolint:gosimple
				return false
			}

			// A consensus state already exists for this height, but it does not match the provided header.
			// The assumption is that Header has already been validated. Thus we can return true as misbehaviour is present
			return true
		}

		// Check that consensus state timestamps are monotonic
		prevCons, prevOk := GetPreviousConsensusState(clientStore, cdc, avaHeader.SubnetHeader.Height)
		nextCons, nextOk := GetNextConsensusState(clientStore, cdc, avaHeader.SubnetHeader.Height)
		// if previous consensus state exists, check consensus state time is greater than previous consensus state time
		// if previous consensus state is not before current consensus state return true
		if prevOk && !prevCons.Timestamp.Before(consState.Timestamp) {
			return true
		}
		// if next consensus state exists, check consensus state time is less than next consensus state time
		// if next consensus state is not after current consensus state return true
		if nextOk && !nextCons.Timestamp.After(consState.Timestamp) {
			return true
		}
	case *Misbehaviour:
		// if heights are equal check that this is valid misbehaviour of a fork
		// otherwise if heights are unequal check that this is valid misbehavior of BFT time violation
		if msg.Header1.SubnetHeader.Height.EQ(msg.Header2.SubnetHeader.Height) {
			blockHashID1 := msg.Header1.SubnetHeader.BlockHash
			blockHashID2 := msg.Header2.SubnetHeader.BlockHash

			if !msg.Header1.PchainHeader.Height.EQ(msg.Header2.PchainHeader.Height) {
				return true
			}

			// Ensure that Commit Hashes are different
			if !bytes.Equal(blockHashID1, blockHashID2) {
				return true
			}
		} else if !msg.Header1.SubnetHeader.Timestamp.After(msg.Header2.SubnetHeader.Timestamp) {
			// Header1 is at greater height than Header2, therefore Header1 time must be less than or equal to
			// Header2 time in order to be valid misbehaviour (violation of monotonic time).
			return true
		}
	}

	return false
}

func (cs *ClientState) verifyMisbehaviour(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, misbehaviour *Misbehaviour) error {
	// Regardless of the type of misbehaviour, ensure that both headers are valid and would have been accepted by light-client

	// Retrieve trusted consensus states for each Header in misbehaviour
	avaConsensusState1, found := GetConsensusState(clientStore, cdc, misbehaviour.Header1.SubnetHeader.Height)
	if !found {
		return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header1 at PrevSubnetHeader.Height: %s", misbehaviour.Header1.SubnetHeader.Height)
	}

	avaConsensusState2, found := GetConsensusState(clientStore, cdc, misbehaviour.Header2.SubnetHeader.Height)
	if !found {
		return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header2 at PrevSubnetHeader.Height: %s", misbehaviour.Header2.SubnetHeader.Height)
	}

	// Check the validity of the two conflicting headers against their respective
	// trusted consensus states
	// NOTE: header height and commitment root assertions are checked in
	// misbehaviour.ValidateBasic by the client keeper and msg.ValidateBasic
	// by the base application.
	if err := checkMisbehaviourHeader(ctx,
		cs, avaConsensusState1, misbehaviour.Header1,
	); err != nil {
		return errorsmod.Wrap(err, "verifying Header1 in Misbehaviour failed")
	}
	if err := checkMisbehaviourHeader(ctx,
		cs, avaConsensusState2, misbehaviour.Header2,
	); err != nil {
		return errorsmod.Wrap(err, "verifying Header2 in Misbehaviour failed")
	}

	return nil
}

// checkMisbehaviourHeader checks that a Header in Misbehaviour is valid misbehaviour given
// a trusted ConsensusState
func checkMisbehaviourHeader(ctx sdk.Context,
	clientState *ClientState, consState *ConsensusState, header *Header,
) error {

	headerUniqVdrs, headerTotalWeight, err := ValidateValidatorSet(ctx, header.Vdrs)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify header")
	}
	consensusUniqVdrs, consensusTotalWeight, err := ValidateValidatorSet(ctx, consState.Vdrs)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify header")
	}
	pchainUniqVdrs, pchainTotalWeight, err := ValidateValidatorSet(ctx, header.SubnetHeader.PchainVdrs)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify header")
	}
	if headerTotalWeight != consensusTotalWeight {
		return errorsmod.Wrap(err, "failed to verify header")
	}
	if headerTotalWeight != pchainTotalWeight {
		return errorsmod.Wrap(err, "failed to verify header")
	}

	if len(headerUniqVdrs) != len(consensusUniqVdrs) {
		return errorsmod.Wrap(err, "failed to verify header")
	}

	if len(headerUniqVdrs) != len(pchainUniqVdrs) {
		return errorsmod.Wrap(err, "failed to verify header")
	}

	for i := range headerUniqVdrs {
		if headerUniqVdrs[i] != consensusUniqVdrs[i] {
			return errorsmod.Wrap(err, "failed to verify header")
		}
	}
	for i := range headerUniqVdrs {
		if headerUniqVdrs[i] != pchainUniqVdrs[i] {
			return errorsmod.Wrap(err, "failed to verify header")
		}
	}

	currentTimestamp := ctx.BlockTime()
	// assert that the age of the trusted consensus state is not older than the trusting period
	if currentTimestamp.Sub(consState.Timestamp) >= clientState.TrustingPeriod {
		return errorsmod.Wrapf(
			ErrTrustingPeriodExpired,
			"current timestamp minus the latest consensus state timestamp is greater than or equal to the trusting period (%d >= %d)",
			currentTimestamp.Sub(consState.Timestamp), clientState.TrustingPeriod,
		)
	}

	chainID := clientState.GetChainID()
	// If chainID is in revision format, then set revision number of chainID with the revision number
	// of the misbehaviour header
	// NOTE: misbehaviour verification is not supported for chains which upgrade to a new chainID without
	// strictly following the chainID revision format
	if clienttypes.IsRevisionFormat(chainID) {
		chainID, _ = clienttypes.SetRevisionNumber(chainID, header.SubnetHeader.Height.RevisionNumber)
	}
	return nil
}
