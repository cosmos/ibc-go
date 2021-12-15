# BEEFY

Beefy is an extension protocol of Grandpa, which seeks to reduce the size of finality proofs for the sole purpose of bridging the Polkadot/Kusama relay chains and parachains to other blockchains capable of following the BEEFY protocol.

The BEEFY protocol consists of an extra round of voting on the MMR root hash of all* finalized Grandpa blocks, by same authority set as Grandpa, for the sole purpose of using merkle-mountain range proofs to prove efficiently which blocks have finality.

BEEFY Leaf data for the MMR is given as:

```rust

/// A MMR leaf versioning scheme.
///
/// Version is a single byte that constist of two components:
/// - `major` - 3 bits
/// - `minor` - 5 bits
///
/// Any change in encoding that adds new items to the structure is considered non-breaking, hence
/// only requires an update of `minor` version. Any backward incompatible change (i.e. decoding to a
/// previous leaf format fails) should be indicated with `major` version bump.
///
/// Given that adding new struct elements in SCALE is backward compatible (i.e. old format can be
/// still decoded, the new fields will simply be ignored). We expect the major version to be bumped
/// very rarely (hopefuly never).
#[derive(Debug, Default, PartialEq, Eq, Clone, Encode, Decode)]
pub struct MmrLeafVersion(u8);

/// A typedef for validator set id.
pub type ValidatorSetId = u64;

/// Details of the next BEEFY authority set.
#[derive(Debug, Default, PartialEq, Eq, Clone, Encode, Decode, TypeInfo)]
pub struct BeefyNextAuthoritySet<MerkleRoot> {
	/// Id of the next set.
	///
	/// Id is required to correlate BEEFY signed commitments with the validator set.
	/// Light Client can easily verify that the commitment witness it is getting is
	/// produced by the latest validator set.
	pub id: crate::ValidatorSetId,
	/// Number of validators in the set.
	///
	/// Some BEEFY Light Clients may use an interactive protocol to verify only subset
	/// of signatures. We put set length here, so that these clients can verify the minimal
	/// number of required signatures.
	pub len: u32,
	/// Merkle Root Hash build from BEEFY AuthorityIds.
	///
	/// This is used by Light Clients to confirm that the commitments are signed by the correct
	/// validator set. Light Clients using interactive protocol, might verify only subset of
	/// signatures, hence don't require the full list here (will receive inclusion proofs).
	pub root: MerkleRoot,
}

/// A standard leaf that gets added every block to the MMR constructed by Substrate's `pallet_mmr`.
#[derive(Debug, PartialEq, Eq, Clone, Encode, Decode)]
pub struct MmrLeaf<BlockNumber, Hash, MerkleRoot> {
	/// Version of the leaf format.
	///
	/// Can be used to enable future format migrations and compatibility.
	/// See [`MmrLeafVersion`] documentation for details.
	pub version: MmrLeafVersion, 
	/// Current block parent number and hash.
	pub parent_number_and_hash: (BlockNumber, Hash),
	/// A merkle root of the next BEEFY authority set.
	pub beefy_next_authority_set: BeefyNextAuthoritySet<MerkleRoot>,
	/// A merkle root of all registered parachain heads.
	pub parachain_heads: MerkleRoot,
}
```


In order to follow BEEFY consensus, light clients must be able to reconstruct this MMR leaf in memory and hash it, then using the MMR root hash and merkle mountain range proofs, check for inclusion of this leaf data in the MMR.


# Step 1. Reconstructing `MMRLeaf`

The relayer that wants to update the state of the light client must provide some additional data to be used to reconstruct the MMRLeaf data for the purpose of hashing and checking for inclusion in the MMR.

 - `version`, this can be gotten from the runtime
 - `parent_number_and_hash`, this can also be gotten from the runtime
 - `beefy_next_authority_set` which can be gotten by:
    ```rust
    // runtime code.
    let beefy_next_authority_set = MmrLeaf::beefy_next_authorities()
     ```
 - `parachain_heads`, this is where things get a bit tricky. Technically, this is the merkle root hash of a `Vec<(ParaId, HeaderHash)>`. But in order to reconstruct this root hash we'd need
   - Our own header hash (This is tricky as well, because we'd like to verify that the messages we're trying to pass across were included in this header, but we'll come back to this.)
   - Merkle proof of inclusion of our own header hash in the root_hash of `parachain_heads`. This can be gotten by:

        ```rust
// runtime code
let para_ids = Paras::parachains();
let mut para_heads: Vec<(u32, Vec<u8>)> = para_ids.iter()
    .filter_map(|id| {
        Paras::para_head(&id).map(|head| (id.into(), head.0)
    });
para_heads.sort();
let own_para_head = para_heads.find(|(id, _)| id == OWN_PARA_ID).unwrap(); // unwrap only if we're sure we have a header in thiblock.
let para_heads = para_heads.into_iter().map(|pair| pair.encode());
let root_hash: H256 = merkle_root(&para_heads);
let proof = merkle_proof(&para_heads, own_para_head);
// such that we should be able to verify 
assert!(merkle_verify(root_hash, proof, own_para_head));

        ```
# Step 2. Using MMR to prove inclusion of leaf data in the MMR root hash
// TODO:


# Step 3. Verify Signatures of 2/3 + 1 BEEFY authorities.
// TODO: 


* Not really all, BEEFY lags behind granpa by a few blocks.