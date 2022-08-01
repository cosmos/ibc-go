package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/ComposableFi/go-merkle-trees/merkle"
	"reflect"

	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/log15"
	"github.com/ComposableFi/go-merkle-trees/hasher"
	merkletypes "github.com/ComposableFi/go-merkle-trees/types"
	"github.com/ComposableFi/go-merkle-trees/mmr"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/ethereum/go-ethereum/crypto"
)

// VerifyClientMessage checks if the clientMessage is of type Header or Misbehaviour and verifies the message
func (cs *ClientState) VerifyClientMessage(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore,
	clientMsg exported.ClientMessage,
) error {
	switch msg := clientMsg.(type) {
	case *Header:
		return cs.verifyHeader(ctx, clientStore, cdc, msg)
	default:
		return clienttypes.ErrInvalidClientType
	}
}

// verifyHeader returns an error if:
// - the client or header provided are not parseable
// - the header is invalid
// - header height is less than or equal to the trusted header height
// - header revision is not equal to trusted header revision
// - header valset commit verification fails
// - header timestamp is past the trusting period in relation to the consensus state
// - header timestamp is less than or equal to the consensus state timestamp
func (cs *ClientState) verifyHeader(
	ctx sdk.Context, clientStore sdk.KVStore, cdc codec.BinaryCodec,
	beefyHeader *Header,
) error {

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
		return ErrCommitmentNotFinal
	}

	if signedCommitment.Commitment.ValidatorSetId != cs.Authority.Id &&
		signedCommitment.Commitment.ValidatorSetId != cs.NextAuthoritySet.Id {
		return ErrAuthoritySetUnknown
	}

	// beefy authorities are signing the hash of the scale-encoded Commitment
	commitmentBytes, err := Encode(&signedCommitment.Commitment)
	if err != nil {
		return sdkerrors.Wrap(err, ErrInvalidCommitment.Error())
	}

	// take keccak hash of the commitment scale-encoded
	commitmentHash := crypto.Keccak256(commitmentBytes)

	// array of leaves in the authority merkle root.
	var authorityLeaves []merkletypes.Leaf

	for i := 0; i < len(signedCommitment.Signatures); i++ {
		signature := signedCommitment.Signatures[i]
		// recover uncompressed public key from signature
		pubkey, err := crypto.SigToPub(commitmentHash, signature.Signature)
		if err != nil {
			return sdkerrors.Wrap(err, ErrInvalidCommitmentSignature.Error())
		}

		// convert public key to ethereum address.
		address := crypto.PubkeyToAddress(*pubkey)
		authorityLeaf := merkletypes.Leaf{
			Hash:  crypto.Keccak256(address[:]),
			Index: uint64(signature.AuthorityIndex),
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
		authoritiesProof := merkle.NewProof(authorityLeaves, authoritiesProof, uint64(cs.Authority.Len), hasher.Keccak256Hasher{})
		valid, err := authoritiesProof.Verify(cs.Authority.AuthorityRoot[:])
		if err != nil || !valid {
			return sdkerrors.Wrap(err, ErrAuthoritySetUnknown.Error())
		}

	// new authority set has kicked in
	case cs.NextAuthoritySet.Id:
		authoritiesProof := merkle.NewProof(authorityLeaves, authoritiesProof, uint64(cs.NextAuthoritySet.Len), hasher.Keccak256Hasher{})
		valid, err := authoritiesProof.Verify(cs.NextAuthoritySet.AuthorityRoot[:])
		if err != nil || !valid {
			return sdkerrors.Wrap(err, ErrAuthoritySetUnknown.Error())
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
					return sdkerrors.Wrap(err, ErrInvalidCommitment.Error())
				}
				// we treat this leaf as the latest leaf in the mmr
				mmrSize := mmr.LeafIndexToMMRSize(mmrUpdateProof.MmrLeafIndex)
				mmrLeaves := []merkletypes.Leaf{
					{
						Hash:  crypto.Keccak256(mmrLeafBytes),
						Index: mmrUpdateProof.MmrLeafIndex,
					},
				}
				mmrProof := mmr.NewProof(mmrSize, mmrUpdateProof.MmrProof, mmrLeaves, hasher.Keccak256Hasher{})
				// verify that the leaf is valid, for the signed mmr-root-hash
				if !mmrProof.Verify(payload.PayloadData[:]) {
					return sdkerrors.Wrap(err, ErrFailedVerifyMMRLeaf.Error()) // error!, mmr proof is invalid
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

	mmrProof, err := cs.parachainHeadersToMMRProof(beefyHeader)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to execute getMMRProf")
	}

	// Given the leaves, we should be able to verify that each parachain header was
	// indeed included in the leaves of our mmr.
	if !mmrProof.Verify(cs.MmrRootHash) {
		root, err := mmrProof.CalculateRoot()
		if err != nil {
			log15.Error(fmt.Sprintf("failed to calculate root for mmr leaf %v", root))
			return sdkerrors.Wrap(err, ErrFailedEncodeMMRLeaf.Error())
		}
		log15.Error(fmt.Sprintf("failed to verify mmr leaf %v", root))
		return sdkerrors.Wrap(err, ErrFailedVerifyMMRLeaf.Error())
	}

	return nil
}

func (cs *ClientState) parachainHeadersToMMRProof(beefyHeader *Header) (*mmr.Proof, error) {
	var mmrLeaves = make([]merkletypes.Leaf, len(beefyHeader.ParachainHeaders))

	// verify parachain headers
	for i := 0; i < len(beefyHeader.ParachainHeaders); i++ {
		// first we need to reconstruct the mmr leaf for this header
		parachainHeader := beefyHeader.ParachainHeaders[i]
		paraIdScale := make([]byte, 4)
		// scale encode para_id
		binary.LittleEndian.PutUint32(paraIdScale[:], parachainHeader.ParaId)
		// scale encode to get parachain heads leaf bytes
		headsLeafBytes := append(paraIdScale, parachainHeader.ParachainHeader...)
		headsLeaf := []merkletypes.Leaf{
			{
				Hash:  crypto.Keccak256(headsLeafBytes),
				Index: uint64(parachainHeader.HeadsLeafIndex),
			},
		}
		parachainHeadsProof := merkle.NewProof(headsLeaf, parachainHeader.ParachainHeadsProof, uint64(parachainHeader.HeadsTotalCount), hasher.Keccak256Hasher{})
		// todo: merkle.Proof.Root() should return fixed bytes
		parachainHeadsRoot, err := parachainHeadsProof.Root()
		// TODO: verify extrinsic root here once trie lib is fixed.
		if err != nil {
			return nil, sdkerrors.Wrap(err, ErrInvalivParachainHeadsProof.Error())
		}

		// not a fan of this but its golang
		var parachainHeads SizedByte32
		copy(parachainHeads[:], parachainHeadsRoot)

		mmrLeaf := BeefyMmrLeaf{
			Version:      parachainHeader.MmrLeafPartial.Version,
			ParentNumber: parachainHeader.MmrLeafPartial.ParentNumber,
			ParentHash:   parachainHeader.MmrLeafPartial.ParentHash,
			BeefyNextAuthoritySet: BeefyAuthoritySet{
				Id:            parachainHeader.MmrLeafPartial.BeefyNextAuthoritySet.Id,
				AuthorityRoot: parachainHeader.MmrLeafPartial.BeefyNextAuthoritySet.AuthorityRoot,
				Len:           parachainHeader.MmrLeafPartial.BeefyNextAuthoritySet.Len,
			},
			ParachainHeads: &parachainHeads,
		}

		// the mmr leaf's are a scale-encoded
		mmrLeafBytes, err := Encode(mmrLeaf)
		if err != nil {
			return nil, sdkerrors.Wrap(err, ErrInvalidMMRLeaf.Error())
		}

		mmrLeaves[i] = merkletypes.Leaf{
			Hash: crypto.Keccak256(mmrLeafBytes),
			// based on our knowledge of the beefy protocol, and the structure of MMRs
			// we are be able to reconstruct the leaf index of this mmr leaf
			// given the parent_number of this leaf, the beefy activation block
			Index: uint64(cs.GetLeafIndexForBlockNumber(parachainHeader.MmrLeafPartial.ParentNumber + 1)),
		}
	}

	mmrProof := mmr.NewProof(beefyHeader.MmrSize, beefyHeader.MmrProofs, mmrLeaves, hasher.Keccak256Hasher{})

	return mmrProof, nil
}

func (cs ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) error {
	beefyHeader, ok := clientMsg.(*Header)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "expected type %T, got %T", &Header{}, beefyHeader)
	}

	consensusStates := make(map[clienttypes.Height]*ConsensusState)

	// iterate over each parachain header and set them in the store.
	for _, v := range beefyHeader.ParachainHeaders {
		// decode parachain header bytes to struct
		header, err := DecodeParachainHeader(v.ParachainHeader)
		if err != nil {
			return sdkerrors.Wrap(err, "failed to decode parachain header")
		}

		// TODO: IBC should allow height to be generic
		height := clienttypes.Height{
			// revion number is used to store paraId
			RevisionNumber: uint64(v.ParaId),
			RevisionHeight: uint64(header.Number),
		}

		// check for duplicate consensus state
		if consensusState, _ := GetConsensusState(clientStore, cdc, height); consensusState != nil {
			// perform no-op
			continue
		}

		trieProof := trie.NewEmptyTrie()
		// load the extrinsics proof which is basically a partial trie
		// that encodes the timestamp extrinsic
		errr := trieProof.LoadFromProof(v.ExtrinsicProof, header.ExtrinsicsRoot[:])
		if errr != nil {
			return sdkerrors.Wrap(err, "failed to load extrinsic proof")
		}
		// the timestamp extrinsic is stored under the key 0u32 in big endian
		key := make([]byte, 4)
		timestamp, err := DecodeExtrinsicTimestamp(trieProof.Get(key))

		if err != nil {
			return sdkerrors.Wrap(err, "failed to decode timestamp extrinsic")
		}

		var ibcCommitmentRoot []byte
		// IBC commitment root is stored in the header digests as a ConsensusItem
		for _, v := range header.Digest {
			if v.IsConsensus {

				consensusID := v.AsConsensus.ConsensusEngineID

				// this is a constant that comes from pallet-ibc
				if bytes.Equal(consensusID[:], []byte("/IBC")) {
					ibcCommitmentRoot = v.AsConsensus.Bytes
				}
			}
		}

		consensusStates[height] = &ConsensusState{
			Timestamp: timestamp,
			Root:      ibcCommitmentRoot,
		}
	}

	// only set consensus states after doing checks
	for height, consensusState := range consensusStates {
		// we store consensus state as (PARA_ID, HEIGHT) => ConsensusState
		setConsensusState(clientStore, cdc, consensusState, height)

		// TODO: pruning!
	}

	setClientState(clientStore, cdc, &cs)

	return nil
}

// CheckForMisbehaviour detects duplicate height misbehaviour and BFT time violation misbehaviour
func (cs ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, msg exported.ClientMessage) bool {
	switch msg := msg.(type) {
	case *Header:
		tmHeader := msg
		consState := tmHeader.ConsensusState()

		// Check if the Client store already has a consensus state for the header's height
		// If the consensus state exists, and it matches the header then we return early
		// since header has already been submitted in a previous UpdateClient.
		prevConsState, _ := GetConsensusState(clientStore, cdc, tmHeader.GetHeight())
		if prevConsState != nil {
			// This header has already been submitted and the necessary state is already stored
			// in client store, thus we can return early without further validation.
			if reflect.DeepEqual(prevConsState, tmHeader.ConsensusState()) {
				return false
			}

			// A consensus state already exists for this height, but it does not match the provided header.
			// The assumption is that Header has already been validated. Thus we can return true as misbehaviour is present
			return true
		}

		// Check that consensus state timestamps are monotonic
		prevCons, prevOk := GetPreviousConsensusState(clientStore, cdc, tmHeader.GetHeight())
		nextCons, nextOk := GetNextConsensusState(clientStore, cdc, tmHeader.GetHeight())
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
		// The correctness of Misbehaviour ClientMessage types is ensured by calling VerifyClientMessage prior to this function
		// Thus, here we can return true, as ClientMessage is of type Misbehaviour
		return true
	}

	return false
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

// GetLeafIndexForBlockNumber given the MmrLeafPartial.ParentNumber & BeefyActivationBlock,
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

// UpdateStateOnMisbehaviour updates state upon misbehaviour, freezing the ClientState. This method should only be called when misbehaviour is detected
// as it does not perform any misbehaviour checks.
func (cs ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, _ exported.ClientMessage) {
	//cs.FrozenHeight = FrozenHeight

	clientStore.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(cdc, &cs))

	panic("implement me")
}
