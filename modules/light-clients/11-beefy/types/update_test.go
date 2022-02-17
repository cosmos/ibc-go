package types_test

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/ComposableFi/go-merkle-trees/merkle"
	"github.com/ComposableFi/go-merkle-trees/mmr"
	client "github.com/ComposableFi/go-substrate-rpc-client/v4"
	clientTypes "github.com/ComposableFi/go-substrate-rpc-client/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
	"github.com/ethereum/go-ethereum/crypto"

	// for creating storage keys
	"github.com/ComposableFi/go-substrate-rpc-client/v4/xxhash"
)

func bytes32(bytes []byte) [32]byte {
	var buffer [32]byte
	copy(buffer[:], bytes)
	return buffer
}

const PARA_ID = 2000

func TestCheckHeaderAndUpdateState(t *testing.T) {

	relayApi, err := client.NewSubstrateAPI("ws://127.0.0.1:65353")
	if err != nil {
		panic(err)
	}
	fmt.Printf("==== connected! ==== \n")

	// _parachainApi, err := client.NewSubstrateAPI("wss://127.0.0.1:9988")
	// if err != nil {
	// 	panic(err)
	// }

	// channel to recieve new SignedCommitments
	ch := make(chan interface{})

	sub, err := relayApi.Client.Subscribe(
		context.Background(),
		"beefy",
		"subscribeJustifications",
		"unsubscribeJustifications",
		"justifications",
		ch,
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("====== subcribed! ======\n")
	var clientState *types.ClientState
	defer sub.Unsubscribe()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				panic("error reading channel")
			}

			compactCommitment := &clientTypes.CompactSignedCommitment{}

			// attempt to decode the SignedCommitments
			err := types.DecodeFromHexString(msg.(string), compactCommitment)
			if err != nil {
				panic(err.Error())
			}

			signedCommitment := compactCommitment.Unpack()

			// latest finalized block number
			blockNumber := uint32(signedCommitment.Commitment.BlockNumber)

			// initialize our client state
			if clientState != nil && clientState.LatestBeefyHeight >= blockNumber {
				fmt.Printf("Skipping stale Commitment for block: %d", signedCommitment.Commitment.BlockNumber)
				continue
			}

			// convert to the blockhash
			blockHash, err := relayApi.RPC.Chain.GetBlockHash(uint64(blockNumber))
			if err != nil {
				panic(err)
			}

			authorities, err := BeefyAuthorities(blockNumber, relayApi, "Authorities")
			if err != nil {
				panic(err)
			}

			nextAuthorities, err := BeefyAuthorities(blockNumber, relayApi, "NextAuthorities")
			if err != nil {
				panic(err)
			}

			var authorityLeaves [][]byte
			for _, v := range authorities {
				hash := crypto.Keccak256(v)
				authorityLeaves = append(authorityLeaves, hash)
			}
			authorityTree, err := merkle.NewTree(types.Keccak256{}).FromLeaves(authorityLeaves)
			if err != nil {
				panic(err)
			}

			var nextAuthorityLeaves [][]byte
			for _, v := range nextAuthorities {
				nextAuthorityLeaves = append(nextAuthorityLeaves, crypto.Keccak256(v))
			}

			nextAuthorityTree, err := merkle.NewTree(types.Keccak256{}).FromLeaves(nextAuthorityLeaves)
			if err != nil {
				panic(err)
			}

			if clientState == nil {
				var authorityTreeRoot = bytes32(authorityTree.Root())
				var nextAuthorityTreeRoot = bytes32(nextAuthorityTree.Root())

				clientState = &types.ClientState{
					MmrRootHash:          signedCommitment.Commitment.Payload[0].Value,
					LatestBeefyHeight:    blockNumber,
					BeefyActivationBlock: 0,
					Authority: &types.BeefyAuthoritySet{
						Id:            uint64(signedCommitment.Commitment.ValidatorSetID),
						Len:           uint32(len(authorities)),
						AuthorityRoot: &authorityTreeRoot,
					},
					NextAuthoritySet: &types.BeefyAuthoritySet{
						Id:            uint64(signedCommitment.Commitment.ValidatorSetID) + 1,
						Len:           uint32(len(nextAuthorities)),
						AuthorityRoot: &nextAuthorityTreeRoot,
					},
				}
				fmt.Printf("\n\nInitializing client state\n\n")
				continue
			}

			// first get all paraIds

			// fetch all registered parachainIds, this method doesn't account for
			// if the parachains whose header was included in the batch of finalized blocks have now
			// lost their parachain slot at this height
			paraIds, err := fetchParaIds(relayApi, blockHash)
			if err != nil {
				panic(err)
			}

			var paraHeaderKeys []clientTypes.StorageKey

			// create full storage key for each known paraId.
			keyPrefix := clientTypes.CreateStorageKeyPrefix("Paras", "Heads")
			// so we can query all blocks from lastfinalized to latestBeefyHeight
			for _, paraId := range paraIds {
				encodedParaId, err := types.Encode(paraId)
				if err != nil {
					panic(err)
				}
				twoxhash := xxhash.New64(encodedParaId).Sum(nil)
				// full key path in the storage source: https://www.shawntabrizi.com/assets/presentations/substrate-storage-deep-dive.pdf
				// xx128("Paras") + xx128("Heads") + xx64(Encode(paraId)) + Encode(paraId)
				fullKey := append(append(keyPrefix, twoxhash[:]...), encodedParaId...)
				paraHeaderKeys = append(paraHeaderKeys, fullKey)
			}
			previousFinalizedHash, err := relayApi.RPC.Chain.GetBlockHash(uint64(clientState.LatestBeefyHeight + 1))
			if err != nil {
				panic(err)
			}

			changeSet, err := relayApi.RPC.State.QueryStorage(paraHeaderKeys, previousFinalizedHash, blockHash)
			if err != nil {
				panic(err)
			}

			// double map that holds block numbers, for which our parachain header
			// was included in the mmr leaf, seeing as our parachain headers might not make it into
			// every relay chain block.
			// Map<BlockNumber, Map<ParaId, Header>>
			var finalizedBlocks = make(map[uint32]map[uint32][]byte)

			// request for batch mmr proof of those leaves
			var leafIndeces []uint64

			for _, changes := range changeSet {
				header, err := relayApi.RPC.Chain.GetHeader(changes.Block)
				if err != nil {
					panic(err)
				}
				var heads = make(map[uint32][]byte)

				for _, keyValue := range changes.Changes {
					if keyValue.HasStorageData {
						var paraId uint32
						err := types.DecodeFromBytes(keyValue.StorageKey[40:], &paraId)
						if err != nil {
							panic(err)
						}

						heads[paraId] = keyValue.StorageData
					}
				}

				// check if heads has target id, else skip
				if heads[PARA_ID] == nil {
					continue
				}

				finalizedBlocks[uint32(header.Number)] = heads

				leafIndeces = append(leafIndeces, uint64(clientState.GetLeafIndexForBlockNumber(uint32(header.Number))))
			}

			// check if finalizedBlocks has a leafIndex for signedCommitment.Commitment.BlockNumber
			// ie check if the latest leaf in the mmr included one of our parachain headers,
			// as we need the latest leaf to construct the MmrUpdateProof.
			// otherwise add it.

			if finalizedBlocks[blockNumber] == nil {
				leafIndeces = append(leafIndeces, uint64(clientState.GetLeafIndexForBlockNumber(blockNumber)))
			}

			// fetch mmr proofs for leaves containing our target paraId
			mmrBatchProof, err := relayApi.RPC.MMR.GenerateBatchProof(leafIndeces, blockHash)
			if err != nil {
				panic(err)
			}


			var parachainHeaders []*types.ParachainHeader

			// track the latest leaf.
			var latestLeaf clientTypes.MmrLeaf

			for _, v := range mmrBatchProof.Leaves {
				var leafBlockNumber = clientState.GetBlockNumberForLeaf(uint32(v.Index))
				if leafBlockNumber == blockNumber {
					// we need this (latest) leaf to construct the MmrUpdateProof
					latestLeaf = v.Leaf
				}
				paraHeaders := finalizedBlocks[leafBlockNumber]
				// the latest mmr leaf doesn't contain our parachain header and as such
				// we don't have a record for this leaf in our finalizedBlocks
				if paraHeaders == nil {
					continue
				}

				var paraHeadsLeaves [][]byte
				// index of our parachain header in the
				// parachain heads merkle root
				var index uint32

				count := 0

				// sort by paraId
				var sortedParaIds []uint32
				for paraId, _ := range paraHeaders {
					sortedParaIds = append(sortedParaIds, paraId)
				}
				sort.SliceStable(sortedParaIds, func(i, j int) bool {
					return sortedParaIds[i] < sortedParaIds[j]
				})

				for _, paraId := range sortedParaIds {
					paraIdScale := make([]byte, 4)
					// scale encode para_id
					binary.LittleEndian.PutUint32(paraIdScale[:], paraId)
					leaf := append(paraIdScale, paraHeaders[paraId]...)
					paraHeadsLeaves = append(paraHeadsLeaves, crypto.Keccak256(leaf))
					if paraId == PARA_ID {
						// note index of paraId
						index = uint32(count)
					}
					count++
				}

				tree, err := merkle.NewTree(types.Keccak256{}).FromLeaves(paraHeadsLeaves)
				if err != nil {
					panic(err)
				}
				paraHeadsProof := tree.Proof([]uint32{index})
				authorityRoot := bytes32(v.Leaf.BeefyNextAuthoritySet.Root[:])
				parentHash := bytes32(v.Leaf.ParentNumberAndHash.Hash[:])

				header := types.ParachainHeader{
					ParachainHeader: paraHeaders[PARA_ID],
					MmrLeafPartial: &types.BeefyMmrLeafPartial{
						Version:      uint8(v.Leaf.Version),
						ParentNumber: uint32(v.Leaf.ParentNumberAndHash.ParentNumber),
						ParentHash:   &parentHash,
						BeefyNextAuthoritySet: types.BeefyAuthoritySet{
							Id:            uint64(v.Leaf.BeefyNextAuthoritySet.ID),
							Len:           uint32(v.Leaf.BeefyNextAuthoritySet.Len),
							AuthorityRoot: &authorityRoot,
						},
					},
					ParachainHeadsProof: paraHeadsProof.ProofHashes(),
					ParaId:              PARA_ID,
					HeadsLeafIndex:      index,
					HeadsTotalCount:     uint32(len(paraHeadsLeaves)),
				}

				parachainHeaders = append(parachainHeaders, &header)
			}

			BeefyNextAuthoritySetRoot := bytes32(latestLeaf.BeefyNextAuthoritySet.Root[:])
			parentHash := bytes32(latestLeaf.ParentNumberAndHash.Hash[:])

			var proofItems [][]byte
			for i := 0; i < len(mmrBatchProof.Proof.Items); i++ {
				proofItems = append(proofItems, mmrBatchProof.Proof.Items[i][:])
			}
			var signatures []*types.CommitmentSignature
			var authorityIndeces []uint32
			// luckily for us, this is already sorted and maps to the right authority index in the authority root.
			for i, v := range signedCommitment.Signatures {
				if v.IsSome() {
					_, sig := v.Unwrap()
					signatures = append(signatures, &types.CommitmentSignature{
						Signature:      sig[:],
						AuthorityIndex: uint32(i),
					})
					authorityIndeces = append(authorityIndeces, uint32(i))
				}
			}

			CommitmentPayload := bytes32(signedCommitment.Commitment.Payload[0].Value)
			ParachainHeads := bytes32(latestLeaf.ParachainHeads[:])
			leafIndex := clientState.GetLeafIndexForBlockNumber(blockNumber)

			mmrUpdateProof := types.MmrUpdateProof{
				MmrLeaf: &types.BeefyMmrLeaf{
					Version:        uint8(latestLeaf.Version),
					ParentNumber:   uint32(latestLeaf.ParentNumberAndHash.ParentNumber),
					ParentHash:     &parentHash,
					ParachainHeads: &ParachainHeads,
					BeefyNextAuthoritySet: types.BeefyAuthoritySet{
						Id:            uint64(latestLeaf.BeefyNextAuthoritySet.ID),
						Len:           uint32(latestLeaf.BeefyNextAuthoritySet.Len),
						AuthorityRoot: &BeefyNextAuthoritySetRoot,
					},
				},
				MmrLeafIndex: uint64(leafIndex),
				MmrProof:     proofItems,
				SignedCommitment: &types.SignedCommitment{
					Commitment: &types.Commitment{
						Payload: &CommitmentPayload,
						// Payload:        []*types.PayloadItem{{PayloadId: []byte("mh"), PayloadData: signedCommitment.Commitment.Payload[:]}},
						BlockNumer:     uint32(signedCommitment.Commitment.BlockNumber),
						ValidatorSetId: uint64(signedCommitment.Commitment.ValidatorSetID),
					},
					Signatures: signatures,
				},
				AuthoritiesProof: authorityTree.Proof(authorityIndeces).ProofHashes(),
			}

			header := types.Header{
				ParachainHeaders: parachainHeaders,
				MmrProofs:        proofItems,
				MmrSize:          mmr.LeafIndexToMMRSize(uint64(leafIndex)),
				MmrUpdateProof:   &mmrUpdateProof,
			}

 			_, _, errs := clientState.CheckHeaderAndUpdateState(sdk.Context{}, nil, nil, &header)
			if errs != nil {
				panic(errs)
			}

			fmt.Printf("\nclientState.LatestBeefyHeight: %d\nclientState.MmrRootHash: %s\n", clientState.LatestBeefyHeight, hex.EncodeToString(clientState.MmrRootHash))

			if clientState.LatestBeefyHeight != uint32(signedCommitment.Commitment.BlockNumber) && !reflect.DeepEqual(clientState.MmrRootHash, signedCommitment.Commitment.Payload) {
				panic("\n\nfailed to update client state!\n")
			}
			fmt.Printf("====== successfully processed justification! ======\n")

			// TODO: assert that the consensus states were actually persisted
			// TODO: tests against invalid proofs and consensus states
		}
	}
}

type Authorities = [][33]uint8

func BeefyAuthorities(blockNumber uint32, conn *client.SubstrateAPI, method string) ([][]byte, error) {
	blockHash, err := conn.RPC.Chain.GetBlockHash(uint64(blockNumber))
	if err != nil {
		return nil, err
	}

	// Fetch metadata
	meta, err := conn.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	storageKey, err := clientTypes.CreateStorageKey(meta, "Beefy", method, nil, nil)
	if err != nil {
		return nil, err
	}

	var authorities Authorities

	ok, err := conn.RPC.State.GetStorage(storageKey, &authorities, blockHash)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("Beefy authorities not found")
	}

	// Convert from ecdsa public key to ethereum address
	var authorityEthereumAddresses [][]byte
	for _, authority := range authorities {
		pub, err := crypto.DecompressPubkey(authority[:])
		if err != nil {
			return nil, err
		}
		ethereumAddress := crypto.PubkeyToAddress(*pub)
		if err != nil {
			return nil, err
		}
		authorityEthereumAddresses = append(authorityEthereumAddresses, ethereumAddress[:])
	}

	return authorityEthereumAddresses, nil
}

func fetchParaIds(conn *client.SubstrateAPI, blockHash clientTypes.Hash) ([]uint32, error) {
	// Fetch metadata
	meta, err := conn.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	storageKey, err := clientTypes.CreateStorageKey(meta, "Paras", "Parachains", nil, nil)
	if err != nil {
		return nil, err
	}

	var paraIds []uint32

	ok, err := conn.RPC.State.GetStorage(storageKey, &paraIds, blockHash)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("Beefy authorities not found")
	}

	return paraIds, nil
}
