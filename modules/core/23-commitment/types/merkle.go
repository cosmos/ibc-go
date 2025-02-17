package types

import (
	"bytes"

	ics23 "github.com/cosmos/ics23/go"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// var representing the proofspecs for an SDK chain
var sdkSpecs = []*ics23.ProofSpec{ics23.IavlSpec, ics23.TendermintSpec}

// ICS 023 Merkle Types Implementation
//
// This file defines Merkle commitment types that implements ICS 023.

// Merkle proof implementation of the Proof interface
// Applied on SDK-based IBC implementation
var _ exported.Root = (*MerkleRoot)(nil)

// GetSDKSpecs is a getter function for the proofspecs of an sdk chain
func GetSDKSpecs() []*ics23.ProofSpec {
	return sdkSpecs
}

// NewMerkleRoot constructs a new MerkleRoot
func NewMerkleRoot(hash []byte) MerkleRoot {
	return MerkleRoot{
		Hash: hash,
	}
}

// GetHash implements RootI interface
func (mr MerkleRoot) GetHash() []byte {
	return mr.Hash
}

// Empty returns true if the root is empty
func (mr MerkleRoot) Empty() bool {
	return len(mr.GetHash()) == 0
}

var _ exported.Prefix = (*MerklePrefix)(nil)

// NewMerklePrefix constructs new MerklePrefix instance
func NewMerklePrefix(keyPrefix []byte) MerklePrefix {
	return MerklePrefix{
		KeyPrefix: keyPrefix,
	}
}

// Bytes returns the key prefix bytes
func (mp MerklePrefix) Bytes() []byte {
	return mp.KeyPrefix
}

// Empty returns true if the prefix is empty
func (mp MerklePrefix) Empty() bool {
	return len(mp.Bytes()) == 0
}

// NewMerklePath creates a new MerklePath instance
// The keys must be passed in from root-to-leaf order.
// NOTE: NewMerklePath returns a commitment/v2 MerklePath.
var NewMerklePath = v2.NewMerklePath

// ApplyPrefix constructs a new commitment path from the arguments. It prepends the prefix key
// with the given path.
func ApplyPrefix(prefix exported.Prefix, path v2.MerklePath) (v2.MerklePath, error) {
	if prefix == nil || prefix.Empty() {
		return v2.MerklePath{}, errorsmod.Wrap(ErrInvalidPrefix, "prefix can't be empty")
	}

	return v2.MerklePath{
		KeyPath: append([][]byte{prefix.Bytes()}, path.KeyPath...),
	}, nil
}

// VerifyMembership verifies the membership of a merkle proof against the given root, path, and value.
// Note that the path is expected as []string{<store key of module>, <key corresponding to requested value>}.
func (proof MerkleProof) VerifyMembership(specs []*ics23.ProofSpec, root exported.Root, path exported.Path, value []byte) error {
	mpath, ok := path.(v2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ErrInvalidProof, "path %v is not of type MerklePath", path)
	}

	if err := validateVerificationArgs(proof, mpath, specs, root); err != nil {
		return err
	}

	// VerifyMembership specific argument validation
	if len(value) == 0 {
		return errorsmod.Wrap(ErrInvalidProof, "empty value in membership proof")
	}

	// Since every proof in chain is a membership proof we can use verifyChainedMembershipProof from index 0
	// to validate entire proof
	return verifyChainedMembershipProof(root.GetHash(), specs, proof.Proofs, mpath, value, 0)
}

// VerifyNonMembership verifies the absence of a merkle proof against the given root and path.
// VerifyNonMembership verifies a chained proof where the absence of a given path is proven
// at the lowest subtree and then each subtree's inclusion is proved up to the final root.
func (proof MerkleProof) VerifyNonMembership(specs []*ics23.ProofSpec, root exported.Root, path exported.Path) error {
	mpath, ok := path.(v2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ErrInvalidProof, "path %v is not of type MerkleProof", path)
	}

	if err := validateVerificationArgs(proof, mpath, specs, root); err != nil {
		return err
	}

	// VerifyNonMembership will verify the absence of key in lowest subtree, and then chain inclusion proofs
	// of all subroots up to final root
	subroot, err := proof.Proofs[0].Calculate()
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidProof, "could not calculate root for proof index 0, merkle tree is likely empty. %v", err)
	}

	key, err := mpath.GetKey(uint64(len(mpath.KeyPath) - 1))
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidProof, "could not retrieve key bytes for key: %s", mpath.KeyPath[len(mpath.KeyPath)-1])
	}

	np := proof.Proofs[0].GetNonexist()
	if np == nil {
		return errorsmod.Wrapf(ErrInvalidProof, "commitment proof must be non-existence proof for verifying non-membership. got: %T", proof.Proofs[0])
	}

	if err := np.Verify(specs[0], subroot, key); err != nil {
		return errorsmod.Wrapf(ErrInvalidProof, "failed to verify non-membership proof with key %s: %v", string(key), err)
	}

	// Verify chained membership proof starting from index 1 with value = subroot
	return verifyChainedMembershipProof(root.GetHash(), specs, proof.Proofs, mpath, subroot, 1)
}

// verifyChainedMembershipProof takes a list of proofs and specs and verifies each proof sequentially ensuring that the value is committed to
// by first proof and each subsequent subroot is committed to by the next subroot and checking that the final calculated root is equal to the given roothash.
// The proofs and specs are passed in from lowest subtree to the highest subtree, but the keys are passed in from highest subtree to lowest.
// The index specifies what index to start chaining the membership proofs, this is useful since the lowest proof may not be a membership proof, thus we
// will want to start the membership proof chaining from index 1 with value being the lowest subroot
func verifyChainedMembershipProof(root []byte, specs []*ics23.ProofSpec, proofs []*ics23.CommitmentProof, keys v2.MerklePath, value []byte, index int) error {
	var (
		subroot []byte
		err     error
	)
	// Initialize subroot to value since the proofs list may be empty.
	// This may happen if this call is verifying intermediate proofs after the lowest proof has been executed.
	// In this case, there may be no intermediate proofs to verify and we just check that lowest proof root equals final root
	subroot = value
	for i := index; i < len(proofs); i++ {
		subroot, err = proofs[i].Calculate()
		if err != nil {
			return errorsmod.Wrapf(ErrInvalidProof, "could not calculate proof root at index %d, merkle tree may be empty. %v", i, err)
		}

		// Since keys are passed in from highest to lowest, we must grab their indices in reverse order
		// from the proofs and specs which are lowest to highest
		key, err := keys.GetKey(uint64(len(keys.KeyPath) - 1 - i))
		if err != nil {
			return errorsmod.Wrapf(ErrInvalidProof, "could not retrieve key bytes for key %s: %v", keys.KeyPath[len(keys.KeyPath)-1-i], err)
		}

		ep := proofs[i].GetExist()
		if ep == nil {
			return errorsmod.Wrapf(ErrInvalidProof, "commitment proof must be existence proof. got: %T at index %d", i, proofs[i])
		}

		// verify membership of the proof at this index with appropriate key and value
		if err := ep.Verify(specs[i], subroot, key, value); err != nil {
			return errorsmod.Wrapf(ErrInvalidProof, "failed to verify membership proof at index %d: %v", i, err)
		}
		// Set value to subroot so that we verify next proof in chain commits to this subroot
		value = subroot
	}

	// Check that chained proof root equals passed-in root
	if !bytes.Equal(root, subroot) {
		return errorsmod.Wrapf(ErrInvalidProof, "proof did not commit to expected root: %X, got: %X. Please ensure proof was submitted with correct proofHeight and to the correct chain.", root, subroot)
	}

	return nil
}

// validateVerificationArgs verifies the proof arguments are valid.
// The merkle path and merkle proof contain a list of keys and their proofs
// which correspond to individual trees. The length of these keys and their proofs
// must equal the length of the given specs. All arguments must be non-empty.
func validateVerificationArgs(proof MerkleProof, path v2.MerklePath, specs []*ics23.ProofSpec, root exported.Root) error {
	if proof.GetProofs() == nil {
		return errorsmod.Wrap(ErrInvalidMerkleProof, "proof must not be empty")
	}

	if root == nil || root.Empty() {
		return errorsmod.Wrap(ErrInvalidMerkleProof, "root cannot be empty")
	}

	if len(specs) != len(proof.Proofs) {
		return errorsmod.Wrapf(ErrInvalidMerkleProof, "length of specs: %d not equal to length of proof: %d", len(specs), len(proof.Proofs))
	}

	if len(path.KeyPath) != len(specs) {
		return errorsmod.Wrapf(ErrInvalidProof, "path length %d not same as proof %d", len(path.KeyPath), len(specs))
	}

	for i, spec := range specs {
		if spec == nil {
			return errorsmod.Wrapf(ErrInvalidProof, "spec at position %d is nil", i)
		}
	}
	return nil
}
