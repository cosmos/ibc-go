package types

import (
	"encoding/binary"
	merkle "github.com/ComposableFi/go-merkle-trees/merkle"
	"github.com/ComposableFi/go-merkle-trees/mmr"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/ethereum/go-ethereum/crypto"
	"reflect"
)

type Keccak256 struct{}

func (b Keccak256) Merge(left, right interface{}) interface{} {
	panic("implement me")
}

// CheckHeaderAndUpdateState checks if the provided header is valid, and if valid it will:
// create the consensus state for the header.Height
// and update the client state if the header height is greater than the latest client state height
// It returns an error if:
// - the client or header provided are not parseable to tendermint types
// - the header is invalid
// - header height is less than or equal to the trusted header height
// - header revision is not equal to trusted header revision
// - header valset commit verification fails
// - header timestamp is past the trusting period in relation to the consensus state
// - header timestamp is less than or equal to the consensus state timestamp
//
// UpdateClient may be used to either create a consensus state for:
// - a future height greater than the latest client state height
// - a past height that was skipped during bisection
// If we are updating to a past height, a consensus state is created for that height to be persisted in client store
// If we are updating to a future height, the consensus state is created and the client state is updated to reflect
// the new latest height
// UpdateClient must only be used to update within a single revision, thus header revision number and trusted height's revision
// number must be the same. To update to a new revision, use a separate upgrade path
// Tendermint client validity checking uses the bisection algorithm described
// in the [Tendermint spec](https://github.com/tendermint/spec/blob/master/spec/consensus/light-client.md).
//
// 1. Reconstruct beefy-go::MmrLeaf using header.parachain_header_proof.mmr_leaf_partial
// 2. specifically ParachainHeads = calculate_root_hash(header.parachain_header_proof, (uint64(2087), parachain_header).encode())
// 3. leaf_hash = keccak hash scale-encode recontructed MmrLeaf
// 4. mmr_root_hash = mmr.calcutate_root_hash(header.parachain_header_proof.mmr_proofs, leaf_hash)
// 5. if client_state.mmr_root_hash == mmr_root_hash
// 5. if header.mmr_update_proof:
// 6. mmr_leaf.parent_number > client_state.relay_parent_number + MAX_BLOCK_WINDOW()
// 7. beefy_go.verifyAuthority(header.mmr_update_proof.signatures, header.mmr_update_proof.authority_proof);
// 8. update client_state.mmr_root_hash = header.mmr_update_proof.mmr_root_hash
//
//Misbehaviour Detection:
// UpdateClient will detect implicit misbehaviour by enforcing certain invariants on any new update call and will return a frozen client.
// 1. Any valid update that creates a different consensus state for an already existing height is evidence of misbehaviour and will freeze client.
// 2. Any valid update that breaks time monotonicity with respect to its neighboring consensus states is evidence of misbehaviour and will freeze client.
//
//Misbehaviour sets frozen height to {0, 1} since it is only used as a boolean value (zero or non-zero).
//
//Pruning:
// UpdateClient will additionally retrieve the earliest consensus state for this clientID and check if it is expired. If it is,
// that consensus state will be pruned from store along with all associated metadata. This will prevent the client store from
// becoming bloated with expired consensus states that can no longer be used for updates and packet verification.
func (cs ClientState) CheckHeaderAndUpdateState(
	_ sdk.Context, _ codec.BinaryCodec, _ sdk.KVStore,
	header exported.Header,
) (exported.ClientState, exported.ConsensusState, error) {
	beefyHeader, ok := header.(*Header)
	if !ok {
		return nil, nil, sdkerrors.Wrapf(
			clienttypes.ErrInvalidHeader, "expected type %T, got %T", &Header{}, header,
		)
	}

	// a new mmr update has arrived
	if beefyHeader.MmrUpdateProof != nil {
		var (
			mmrUpdateProof   = beefyHeader.MmrUpdateProof
			proof            = beefyHeader.MmrUpdateProof.AuthoritiesProof
			signedCommitment = beefyHeader.MmrUpdateProof.SignedCommitment
		)

		// take keccak hash of the commitment scale-encoded
		commitmentHash := crypto.Keccak256(signedCommitment.Commitment.encode()) // todo: implement encode

		// array of leaves in the authority merkle root.
		var authorityLeaves []merkle.Leaf

		for _, signature := range signedCommitment.Signatures {
			// recover uncompressed public key from signature
			pubkey, _ := crypto.Ecrecover(commitmentHash, signature.Signature)
			// convert public key to ethereum address.
			address := crypto.Keccak256(pubkey[1:])[12:]
			authorityLeaf := merkle.Leaf{
				Hash:  crypto.Keccak256(address),
				Index: signature.AuthorityIndex,
			}
			authorityLeaves = append(authorityLeaves, authorityLeaf)
		}

		// flag for if the authority set has been updated
		updatedAuthority := false
		// assert that known authorities signed this commitment
		switch signedCommitment.Commitment.ValidatorSetId {
		case cs.Authority.Id:
			authoritiesProof := merkle.NewProof(authorityLeaves, proof, cs.Authority.Len, Keccak256{})
			if !reflect.DeepEqual(authoritiesProof.Root(), cs.Authority.AuthorityRoot) {
				return nil, nil, nil
			}
		// new authority set has kicked in
		case cs.NextAuthoritySet.Id:
			authoritiesProof := merkle.NewProof(authorityLeaves, proof, cs.NextAuthoritySet.Len, Keccak256{})
			if !reflect.DeepEqual(authoritiesProof.Root(), cs.NextAuthoritySet.AuthorityRoot) {
				return nil, nil, nil
			}
			updatedAuthority = true
		}

		// only update if we have a higher block number.
		if signedCommitment.Commitment.BlockNumer > cs.LatestBeefyHeight {
			for _, payload := range signedCommitment.Commitment.Payload {
				// checks for the right payloadId
				if reflect.DeepEqual(payload.PayloadId, []byte("mh")) {
					// the next authorities are in the latest BeefyMmrLeaf
					mmrLeafBytes, err := mmrUpdateProof.MmrLeaf.encode()
					// we treat this leaf as the latest leaf in the mmr
					mmrSize := mmr.LeafIndexToMMRSize(mmrUpdateProof.MmrLeafIndex)
					mmrLeaves := []mmr.Leaf{
						{
							Hash:  crypto.Keccak256(mmrLeafBytes),
							Index: mmrUpdateProof.MmrLeafIndex,
						},
					}
					mmrProof := mmr.NewProof(mmrSize, mmrUpdateProof.MmrProof, mmrLeaves, Keccak256{})
					// verify that the leaf is valid, for the signed mmr-root-hash
					if !mmrProof.Verify(payload.PayloadData) {
						return nil, nil, err // error!, mmr proof is invalid
					}
					// update the block_number
					cs.LatestBeefyHeight = signedCommitment.Commitment.BlockNumer
					// updates the mmr_root_hash
					cs.MmrRootHash = payload.PayloadData
					// authority set has changed, rotate our view of the authorities
					if updatedAuthority {
						cs.Authority = cs.NextAuthoritySet
						// mmr leaf has been verified, use it to update our view of the next authority set
						cs.NextAuthoritySet = &mmrUpdateProof.MmrLeaf.BeefyNextAuthoritySet
					}
					break
				}
			}
		}
	}

	var mmrLeaves []mmr.Leaf

	// verify parachain headers
	for _, parachainHeader := range beefyHeader.ParachainHeaders {
		paraIdScale := make([]byte, 4)
		// scale encode para_id
		binary.LittleEndian.PutUint32(paraIdScale[:], parachainHeader.ParaId)

		//cbmt := merkle.CBMT{Merge: Blake256{}}
		headsLeaf := []merkle.Leaf{
			merkle.Leaf{
				Hash:  crypto.Keccak256(append(paraIdScale, parachainHeader.ParachainHeader...)),
				Index: parachainHeader.HeadsLeafIndex,
			},
		}
		parachainHeadsProof, _ := merkle.NewProof(headsLeaf, parachainHeader.ParachainHeadsProof, cs.Authority.Len)

		mmrLeafBytes, err := BeefyMmrLeaf {
			Version:        parachainHeader.MmrLeafPartial.Version,
			ParentNumber:   parachainHeader.MmrLeafPartial.ParentNumber,
			ParentHash:     parachainHeader.MmrLeafPartial.ParentHash,
			ParachainHeads: parachainHeadsProof.Root(),
			BeefyNextAuthoritySet: BeefyAuthoritySet{
				Id:            cs.NextAuthoritySet.Id,
				AuthorityRoot: cs.NextAuthoritySet.AuthorityRoot,
				Len:           cs.NextAuthoritySet.Len,
			},
		}.Encode()

		mmrData := mmr.Leaf{
			Hash:  crypto.Keccak256(mmrLeafBytes),
			Index: parachainHeader.MmrLeafIndex,
		}

		mmrLeaves = append(mmrLeaves, mmrData)
	}

	mmrProof := mmr.NewProof(beefyHeader.MmrSize, beefyHeader.MmrProofs, mmrLeaves, Keccak256{})

	if !mmrProof.Verify(cs.MmrRootHash) {
		return nil, nil, nil // error!, mmr proof is invalid
	}

	// todo: set consensus state for every given header.

	// // reject duplicate headers
	// // Check if the Client store already has a consensus state for the header's height
	// // If the consensus state exists, and it matches the header then we return early
	// // since header has already been submitted in a previous UpdateClient.
	//var conflictingHeader bool
	//prevConsState, _ := GetConsensusState(clientStore, cdc, header.GetHeight())
	//if prevConsState != nil {
	//	// This header has already been submitted and the necessary state is already stored
	//	// in client store, thus we can return early without further validation.
	//	if reflect.DeepEqual(prevConsState, tmHeader.ConsensusState()) {
	//		return &cs, prevConsState, nil
	//	}
	//	// A consensus state already exists for this height, but it does not match the provided header.
	//	// Thus, we must check that this header is valid, and if so we will freeze the client.
	//	conflictingHeader = true
	//}

	// // get consensus state from clientStore
	// trustedConsState, err := GetConsensusState(clientStore, cdc, tmHeader.TrustedHeight)
	// if err != nil {
	// 	return nil, nil, sdkerrors.Wrapf(
	// 		err, "could not get consensus state from clientstore at TrustedHeight: %s", tmHeader.TrustedHeight,
	// 	)
	// }

	// if err := verifyParachainLeaf(&cs, trustedConsState, tmHeader, ctx.BlockTime()); err != nil {
	// 	return nil, nil, err
	// }

	// // forks
	// consState := tmHeader.ConsensusState()
	// // Header is different from existing consensus state and also valid, so freeze the client and return
	// if conflictingHeader {
	// 	cs.FrozenHeight = FrozenHeight
	// 	return &cs, consState, nil
	// }

	// // Check that consensus state timestamps are monotonic
	// prevCons, prevOk := GetPreviousConsensusState(clientStore, cdc, header.GetHeight())
	// nextCons, nextOk := GetNextConsensusState(clientStore, cdc, header.GetHeight())
	// // if previous consensus state exists, check consensus state time is greater than previous consensus state time
	// // if previous consensus state is not before current consensus state, freeze the client and return.
	// if prevOk && !prevCons.Timestamp.Before(consState.Timestamp) {
	// 	cs.FrozenHeight = FrozenHeight
	// 	return &cs, consState, nil
	// }
	// // if next consensus state exists, check consensus state time is less than next consensus state time
	// // if next consensus state is not after current consensus state, freeze the client and return.
	// if nextOk && !nextCons.Timestamp.After(consState.Timestamp) {
	// 	cs.FrozenHeight = FrozenHeight
	// 	return &cs, consState, nil
	// }

	// // pruning
	// // Check the earliest consensus state to see if it is expired, if so then set the prune height
	// // so that we can delete consensus state and all associated metadata.
	// var (
	// 	pruneHeight exported.Height
	// 	pruneError  error
	// )
	// pruneCb := func(height exported.Height) bool {
	// 	consState, err := GetConsensusState(clientStore, cdc, height)
	// 	// this error should never occur
	// 	if err != nil {
	// 		pruneError = err
	// 		return true
	// 	}
	// 	if cs.IsExpired(consState.Timestamp, ctx.BlockTime()) {
	// 		pruneHeight = height
	// 	}
	// 	return true
	// }
	// IterateConsensusStateAscending(clientStore, pruneCb)
	// if pruneError != nil {
	// 	return nil, nil, pruneError
	// }
	// // if pruneHeight is set, delete consensus state and metadata
	// if pruneHeight != nil {
	// 	deleteConsensusState(clientStore, pruneHeight)
	// 	deleteConsensusMetadata(clientStore, pruneHeight)
	// }

	// newClientState, consensusState := update(ctx, clientStore, &cs, tmHeader)
	// return newClientState, consensusState, nil
}

// checkTrustedHeader checks that consensus state matches trusted fields of Header
// func checkTrustedHeader(header *Header, consState *ConsensusState) error {
// 	tmTrustedValidators, err := tmtypes.ValidatorSetFromProto(header.TrustedValidators)
// 	if err != nil {
// 		return sdkerrors.Wrap(err, "trusted validator set in not tendermint validator set type")
// 	}

// 	// assert that trustedVals is NextValidators of last trusted header
// 	// to do this, we check that trustedVals.Hash() == consState.NextValidatorsHash
// 	tvalHash := tmTrustedValidators.Hash()
// 	if !bytes.Equal(consState.NextValidatorsHash, tvalHash) {
// 		return sdkerrors.Wrapf(
// 			ErrInvalidValidatorSet,
// 			"trusted validators %s, does not hash to latest trusted validators. Expected: %X, got: %X",
// 			header.TrustedValidators, consState.NextValidatorsHash, tvalHash,
// 		)
// 	}
// 	return nil
// }

// checkValidity checks if the Tendermint header is valid.
// CONTRACT: consState.Height == header.TrustedHeight
// func checkValidity(
// 	clientState *ClientState, consState *ConsensusState,
// 	header *Header, currentTimestamp time.Time,
// ) error {
// 	if err := checkTrustedHeader(header, consState); err != nil {
// 		return err
// 	}

// 	// UpdateClient only accepts updates with a header at the same revision
// 	// as the trusted consensus state
// 	if header.GetHeight().GetRevisionNumber() != header.TrustedHeight.RevisionNumber {
// 		return sdkerrors.Wrapf(
// 			ErrInvalidHeaderHeight,
// 			"header height revision %d does not match trusted header revision %d",
// 			header.GetHeight().GetRevisionNumber(), header.TrustedHeight.RevisionNumber,
// 		)
// 	}

// 	tmTrustedValidators, err := tmtypes.ValidatorSetFromProto(header.TrustedValidators)
// 	if err != nil {
// 		return sdkerrors.Wrap(err, "trusted validator set in not tendermint validator set type")
// 	}

// 	tmSignedHeader, err := tmtypes.SignedHeaderFromProto(header.SignedHeader)
// 	if err != nil {
// 		return sdkerrors.Wrap(err, "signed header in not tendermint signed header type")
// 	}

// 	tmValidatorSet, err := tmtypes.ValidatorSetFromProto(header.ValidatorSet)
// 	if err != nil {
// 		return sdkerrors.Wrap(err, "validator set in not tendermint validator set type")
// 	}

// 	// assert header height is newer than consensus state
// 	if header.GetHeight().LTE(header.TrustedHeight) {
// 		return sdkerrors.Wrapf(
// 			clienttypes.ErrInvalidHeader,
// 			"header height ≤ consensus state height (%s ≤ %s)", header.GetHeight(), header.TrustedHeight,
// 		)
// 	}

// 	chainID := clientState.GetChainID()
// 	// If chainID is in revision format, then set revision number of chainID with the revision number
// 	// of the header we are verifying
// 	// This is useful if the update is at a previous revision rather than an update to the latest revision
// 	// of the client.
// 	// The chainID must be set correctly for the previous revision before attempting verification.
// 	// Updates for previous revisions are not supported if the chainID is not in revision format.
// 	if clienttypes.IsRevisionFormat(chainID) {
// 		chainID, _ = clienttypes.SetRevisionNumber(chainID, header.GetHeight().GetRevisionNumber())
// 	}

// 	// Construct a trusted header using the fields in consensus state
// 	// Only Height, Time, and NextValidatorsHash are necessary for verification
// 	trustedHeader := tmtypes.Header{
// 		ChainID:            chainID,
// 		Height:             int64(header.TrustedHeight.RevisionHeight),
// 		Time:               consState.Timestamp,
// 		NextValidatorsHash: consState.NextValidatorsHash,
// 	}
// 	signedHeader := tmtypes.SignedHeader{
// 		Header: &trustedHeader,
// 	}

// 	// Verify next header with the passed-in trustedVals
// 	// - asserts trusting period not passed
// 	// - assert header timestamp is not past the trusting period
// 	// - assert header timestamp is past latest stored consensus state timestamp
// 	// - assert that a TrustLevel proportion of TrustedValidators signed new Commit
// 	err = light.Verify(
// 		&signedHeader,
// 		tmTrustedValidators, tmSignedHeader, tmValidatorSet,
// 		clientState.TrustingPeriod, currentTimestamp, clientState.MaxClockDrift, clientState.TrustLevel.ToTendermint(),
// 	)
// 	if err != nil {
// 		return sdkerrors.Wrap(err, "failed to verify header")
// 	}
// 	return nil
// }

// update the consensus state from a new header and set processed time metadata
// func update(ctx sdk.Context, clientStore sdk.KVStore, clientState *ClientState, header *Header) (*ClientState, *ConsensusState) {
// 	height := header.GetHeight().(clienttypes.Height)
// 	if height.GT(clientState.LatestHeight) {
// 		clientState.LatestHeight = height
// 	}
// 	consensusState := &ConsensusState{
// 		Timestamp:          header.GetTime(),
// 		Root:               commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()),
// 		NextValidatorsHash: header.Header.NextValidatorsHash,
// 	}

// 	// set metadata for this consensus state
// 	setConsensusMetadata(ctx, clientStore, header.GetHeight())

// 	return clientState, consensusState
// }
