package types

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ComposableFi/go-merkle-trees/merkle"
	"github.com/ComposableFi/go-merkle-trees/mmr"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/ethereum/go-ethereum/crypto"
)

type Keccak256 struct{}

func (b Keccak256) Merge(left, right interface{}) interface{} {
	l := left.([]byte)
	r := right.([]byte)
	return crypto.Keccak256(append(l, r...))
}

func (b Keccak256) Hash(data []byte) ([]byte, error) {
	return crypto.Keccak256(data), nil
}

// CheckHeaderAndUpdateState checks if the provided header(s) is valid, and if valid it will:
// create the consensus states for the header.Height
// and update the client state if the header height is greater than the latest client state height
// It returns an error if:
// - the client or header provided are not parseable to beefy types
// - the header is invalid
// - beefy valset commitment verification fails
// - header timestamp is less than or equal to the consensus state timestamp
//
// UpdateClient may be used to either create a consensus state for:
// - a future height greater than the latest client state height
// - a past height that was skipped during bisection
// If we are updating to a past height, a consensus state is created for that height to be persisted in client store
// If we are updating to a future height, the consensus state is created and the client state is updated to reflect
// the new latest height
//
// 1. Reconstruct MmrLeaf using header.parachain_header_proof.mmr_leaf_partial
// 2. specifically ParachainHeads = calculate_root_hash(header.parachain_header_proof, (uint64(2087), parachain_header).encode())
// 3. leaf_hash = keccak hash scale-encode recontructed MmrLeaf
// 4. mmr_root_hash = mmr.calcutate_root_hash(header.parachain_header_proof.mmr_proofs, leaf_hash)
// 5. if client_state.mmr_root_hash == mmr_root_hash
// 5. if header.mmr_update_proof:
// 6. mmr_leaf.parent_number > client_state.relay_parent_number + MAX_BLOCK_WINDOW()
// 7. beefy_go.verifyAuthority(header.mmr_update_proof.signatures, header.mmr_update_proof.authority_proof);
// 8. update client_state.mmr_root_hash = header.mmr_update_proof.mmr_root_hash
//
//TODO: Misbehaviour Detection:
//Pruning:
// UpdateClient will additionally retrieve the earliest consensus state for this clientID and check if it is expired. If it is,
// that consensus state will be pruned from store along with all associated metadata. This will prevent the client store from
// becoming bloated with expired consensus states that can no longer be used for updates and packet verification.
func (cs *ClientState) CheckHeaderAndUpdateState(
	_ sdk.Context, _ codec.BinaryCodec, _ sdk.KVStore, header exported.Header,
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
			authoritiesProof = beefyHeader.MmrUpdateProof.AuthoritiesProof
			signedCommitment = beefyHeader.MmrUpdateProof.SignedCommitment
		)

		// checking signatures is expensive (667 authorities for kusama),
		// we want to know if these sigs meet the minimum threshold before proceeding
		// and are by a known authority set (the current one, or the next one)
		if authoritiesThreshold(*cs.Authority) > uint32(len(signedCommitment.Signatures)) ||
			authoritiesThreshold(*cs.NextAuthoritySet) > uint32(len(signedCommitment.Signatures)) {
			// todo: error commitment isn't final
			return nil, nil, nil
		}

		if signedCommitment.Commitment.ValidatorSetId != cs.Authority.Id &&
			signedCommitment.Commitment.ValidatorSetId != cs.NextAuthoritySet.Id {
			// todo: authority set is unknown
			return nil, nil, nil
		}

		// beefy authorities are signing the hash of the scale-encoded Commitment
		commitmentBytes, err := Encode(&signedCommitment.Commitment)
		if err != nil {
			// todo: proper errors
			return nil, nil, err
		}

		// take keccak hash of the commitment scale-encoded
		commitmentHash := crypto.Keccak256(commitmentBytes)

		// array of leaves in the authority merkle root.
		var authorityLeaves []merkle.Leaf

		for i := 0; i < len(signedCommitment.Signatures); i++ {
			signature := signedCommitment.Signatures[i]
			// recover uncompressed public key from signature
			pubkey, err := crypto.SigToPub(commitmentHash, signature.Signature)

			if err != nil {
				// todo: error failed to recover signature!
				return nil, nil, err
			}

			// convert public key to ethereum address.
			address := crypto.PubkeyToAddress(*pubkey)
			authorityLeaf := merkle.Leaf{
				Hash:  crypto.Keccak256(address[:]),
				Index: signature.AuthorityIndex,
			}
			authorityLeaves = append(authorityLeaves, authorityLeaf)
		}

		// flag for if the authority set has been updated
		updatedAuthority := false

		// assert that known authorities signed this commitment, only 2 cases because we already
		// made a prior check to assert that authorities are known
		switch signedCommitment.Commitment.ValidatorSetId {
		case cs.Authority.Id:
			// here we construct a merkle proof, and verify that the public keys which produced this signature
			// are part of the current round.
			authoritiesProof := merkle.NewProof(authorityLeaves, authoritiesProof, cs.Authority.Len, Keccak256{})
			valid, err := authoritiesProof.Verify(cs.Authority.AuthorityRoot[:])
			if err != nil || !valid {
				// todo: error unknown authority set!
				return nil, nil, nil
			}

		// new authority set has kicked in
		case cs.NextAuthoritySet.Id:
			authoritiesProof := merkle.NewProof(authorityLeaves, authoritiesProof, cs.NextAuthoritySet.Len, Keccak256{})
			valid, err := authoritiesProof.Verify(cs.NextAuthoritySet.AuthorityRoot[:])
			if err != nil || !valid {
				// todo: error unknown authority set!
				return nil, nil, nil
			}
			updatedAuthority = true
		}

		// only update if we have a higher block number.
		if signedCommitment.Commitment.BlockNumer > cs.LatestBeefyHeight {
			for _, payload := range signedCommitment.Commitment.Payload {
				mmrRootId := []byte("mh")
				// checks for the right payloadId
				if bytes.Equal(payload.PayloadId[:], mmrRootId) {
					// the next authorities are in the latest BeefyMmrLeaf

					// scale encode the mmr leaf
					mmrLeafBytes, err := Encode(mmrUpdateProof.MmrLeaf)
					if err != nil {
						return nil, nil, err
					}
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
					if !mmrProof.Verify(payload.PayloadData[:]) {
						return nil, nil, err // error!, mmr proof is invalid
					}
					// update the block_number
					cs.LatestBeefyHeight = signedCommitment.Commitment.BlockNumer
					// updates the mmr_root_hash
					cs.MmrRootHash = payload.PayloadData[:]
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

	var mmrLeaves = make([]mmr.Leaf, len(beefyHeader.ParachainHeaders))
	// for debug purposes
	var mmrLeavesRaw = make([]BeefyMmrLeaf, len(beefyHeader.ParachainHeaders))

	// verify parachain headers
	for i := 0; i < len(beefyHeader.ParachainHeaders); i++ {
		// first we need to reconstruct the mmr leaf for this header
		parachainHeader := beefyHeader.ParachainHeaders[i]
		paraIdScale := make([]byte, 4)
		// scale encode para_id
		binary.LittleEndian.PutUint32(paraIdScale[:], parachainHeader.ParaId)
		// scale encode to get parachain heads leaf bytes
		headsLeafBytes := append(paraIdScale, parachainHeader.ParachainHeader...)
		headsLeaf := []merkle.Leaf{
			{
				Hash:  crypto.Keccak256(headsLeafBytes),
				Index: parachainHeader.HeadsLeafIndex,
			},
		}
		parachainHeadsProof := merkle.NewProof(headsLeaf, parachainHeader.ParachainHeadsProof, parachainHeader.HeadsTotalCount, Keccak256{})
		// todo: merkle.Proof.Root() should return fixed bytes
		ParachainHeadsRoot, err := parachainHeadsProof.Root()
		// not a fan of this but its golang
		var ParachainHeads [32]byte
		copy(ParachainHeads[:], ParachainHeadsRoot)

		if err != nil {
			// todo: invalid parachain heads proof!
			return nil, nil, err
		}

		mmrLeaf := BeefyMmrLeaf{
			Version:      parachainHeader.MmrLeafPartial.Version,
			ParentNumber: parachainHeader.MmrLeafPartial.ParentNumber,
			ParentHash:   parachainHeader.MmrLeafPartial.ParentHash,
			BeefyNextAuthoritySet: BeefyAuthoritySet{
				Id:            cs.NextAuthoritySet.Id,
				AuthorityRoot: cs.NextAuthoritySet.AuthorityRoot,
				Len:           cs.NextAuthoritySet.Len,
			},
			ParachainHeads: &ParachainHeads,
		}
		// for debug purposes
		mmrLeavesRaw[i] = mmrLeaf

		// the mmr leafs are a scale-encoded
		mmrLeafBytes, err := Encode(mmrLeaf)
		if err != nil {
			// todo: error failed to encode MmrLeaf
			return nil, nil, err
		}

		mmrLeaves[i] = mmr.Leaf{
			Hash: crypto.Keccak256(mmrLeafBytes),
			// based on our knowledge of the beefy protocol, and the structure of mmrs
			// we are be able to reconstruct the leaf index of this mmr leaf
			// given the parent_number of this leaf, the beefy activation block
			Index: uint64(cs.GetLeafIndexForBlockNumber(parachainHeader.MmrLeafPartial.ParentNumber + 1)),
		}
	}

	mmrProof := mmr.NewProof(beefyHeader.MmrSize, beefyHeader.MmrProofs, mmrLeaves, Keccak256{})

	// Given the leaves, we should be able to verify that each parachain header was
	// indeed included in the leaves of our mmr.
	if !mmrProof.Verify(cs.MmrRootHash) {
		root, err := mmrProof.CalculateRoot()
		if err != nil {
			// todo: error failed to encode MmrLeaf
			panic(err)

		}
		fmt.Printf("failed to verify mmr leaf %v", root)
		return nil, nil, nil // error!, mmr proof is invalid
	}

	// todo: set consensus state for every given header.
	// reject duplicate headers
	// Check if the Client store already has a consensus state for the header's height
	// If the consensus state exists, and it matches the header then we return early
	// since header has already been submitted in a previous UpdateClient.
	// for _, header := range beefyHeader.ParachainHeaders {
	// 	var conflictingHeader bool
	// 	prevConsState, _ := GetConsensusState(clientStore, cdc, header.ParachainHeader)
	// 	if prevConsState != nil {
	// 		// This header has already been submitted and the necessary state is already stored
	// 		// in client store, thus we can return early without further validation.
	// 		if reflect.DeepEqual(prevConsState, header.ConsensusState()) {
	// 			return &cs, prevConsState, nil
	// 		}
	// 		// A consensus state already exists for this height, but it does not match the provided header.
	// 		// Thus, we must check that this header is valid, and if so we will freeze the client.
	// 		conflictingHeader = true
	// 	}

	// 	// get consensus state from clientStore
	// 	trustedConsState, err := GetConsensusState(clientStore, cdc, tmHeader.TrustedHeight)
	// 	if err != nil {
	// 		return nil, nil, sdkerrors.Wrapf(
	// 			err, "could not get consensus state from clientstore at TrustedHeight: %s", tmHeader.TrustedHeight,
	// 		)
	// 	}
	// 	// forks
	// 	consState := tmHeader.ConsensusState()
	// 	// Header is different from existing consensus state and also valid, so freeze the client and return
	// 	if conflictingHeader {
	// 		cs.FrozenHeight = FrozenHeight
	// 		return &cs, consState, nil
	// 	}

	// 	// Check that consensus state timestamps are monotonic
	// 	prevCons, prevOk := GetPreviousConsensusState(clientStore, cdc, header.GetHeight())
	// 	nextCons, nextOk := GetNextConsensusState(clientStore, cdc, header.GetHeight())
	// 	// if previous consensus state exists, check consensus state time is greater than previous consensus state time
	// 	// if previous consensus state is not before current consensus state, freeze the client and return.
	// 	if prevOk && !prevCons.Timestamp.Before(consState.Timestamp) {
	// 		cs.FrozenHeight = FrozenHeight
	// 		return &cs, consState, nil
	// 	}
	// 	// if next consensus state exists, check consensus state time is less than next consensus state time
	// 	// if next consensus state is not after current consensus state, freeze the client and return.
	// 	if nextOk && !nextCons.Timestamp.After(consState.Timestamp) {
	// 		cs.FrozenHeight = FrozenHeight
	// 		return &cs, consState, nil
	// 	}
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
	return nil, nil, nil
}

func (cs ClientState) GetBlockNumberForLeaf(leafIndex uint32) uint32 {
	var blockNumber uint32

	// calculate the leafIndex for this leaf.
	if cs.BeefyActivationBlock == 0 {
		// in this case the leaf index is the same as the block number - 1 (leaf index starts at 0)
		blockNumber = leafIndex + 1
	} else {
		// in this case the leaf index is activation block - current block number.
		blockNumber = cs.BeefyActivationBlock + leafIndex
	}

	return blockNumber
}

// given the MmrLeafPartial.ParentNumber & BeefyActivationBlock,
func (cs ClientState) GetLeafIndexForBlockNumber(blockNumber uint32) uint32 {
	var leafIndex uint32

	// calculate the leafIndex for this leaf.
	if cs.BeefyActivationBlock == 0 {
		// in this case the leaf index is the same as the block number - 1 (leaf index starts at 0)
		leafIndex = blockNumber - 1
	} else {
		// in this case the leaf index is activation block - current block number.
		leafIndex = cs.BeefyActivationBlock - (blockNumber + 1)
	}

	return leafIndex
}

func authoritiesThreshold(authoritySet BeefyAuthoritySet) uint32 {
	return 2*authoritySet.Len/3 + 1
}
