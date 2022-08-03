package types_test

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"testing"

	// types2 "github.com/cosmos/cosmos-sdk/x/params/types"
	// "github.com/cosmos/ibc-go/v3/modules/core/02-client/keeper"

	"github.com/stretchr/testify/require"

	"github.com/ComposableFi/go-merkle-trees/hasher"
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

var (
	BEEFY_TEST_MODE    = os.Getenv("BEEFY_TEST_MODE")
	RPC_CLIENT_ADDRESS = os.Getenv("RPC_CLIENT_ADDRESS")
	UPDATE_STATE_MODE  = os.Getenv("UPDATE_STATE_MODE")
)

func bytes32(bytes []byte) types.SizedByte32 {
	var buffer types.SizedByte32
	copy(buffer[:], bytes)
	return buffer
}

const PARA_ID uint32 = 2000

func TestCheckHeaderAndUpdateState(t *testing.T) {
	// if BEEFY_TEST_MODE != "true" {
	// 	t.Skip("skipping test in short mode")
	// }
	if RPC_CLIENT_ADDRESS == "" {
		t.Log("==== RPC_CLIENT_ADDRESS not set, will use default ==== ")
		RPC_CLIENT_ADDRESS = "ws://127.0.0.1:9944"
	}

	relayApi, err := client.NewSubstrateAPI(RPC_CLIENT_ADDRESS)
	require.NoError(t, err)

	t.Log("==== connected! ==== ")

	// _parachainApi, err := client.NewSubstrateAPI("wss://127.0.0.1:9988")
	// if err != nil {
	// 	panic(err)
	// }

	// channel to receive new SignedCommitments
	ch := make(chan interface{})

	sub, err := relayApi.Client.Subscribe(
		context.Background(),
		"beefy",
		"subscribeJustifications",
		"unsubscribeJustifications",
		"justifications",
		ch,
	)
	require.NoError(t, err)

	t.Log("====== subcribed! ======")
	var clientState *types.ClientState
	defer sub.Unsubscribe()

	for count := 0; count < 100; count++ {
		select {
		case msg, ok := <-ch:
			require.True(t, ok, "error reading channel")

			compactCommitment := clientTypes.CompactSignedCommitment{}

			// attempt to decode the SignedCommitments
			err = types.DecodeFromHexString(msg.(string), &compactCommitment)
			require.NoError(t, err)

			signedCommitment := compactCommitment.Unpack()

			// latest finalized block number
			blockNumber := uint32(signedCommitment.Commitment.BlockNumber)

			// initialize our client state
			if clientState != nil && clientState.LatestBeefyHeight >= blockNumber {
				t.Logf("Skipping stale Commitment for block: %d", signedCommitment.Commitment.BlockNumber)
				continue
			}

			// convert to the blockHash
			blockHash, err := relayApi.RPC.Chain.GetBlockHash(uint64(blockNumber))
			require.NoError(t, err)

			authorities, err := BeefyAuthorities(blockNumber, relayApi, "Authorities")
			require.NoError(t, err)

			nextAuthorities, err := BeefyAuthorities(blockNumber, relayApi, "NextAuthorities")
			require.NoError(t, err)

			var authorityLeaves [][]byte
			for _, v := range authorities {
				authorityLeaves = append(authorityLeaves, crypto.Keccak256(v))
			}

			authorityTree, err := merkle.NewTree(hasher.Keccak256Hasher{}).FromLeaves(authorityLeaves)
			require.NoError(t, err)

			var nextAuthorityLeaves [][]byte
			for _, v := range nextAuthorities {
				nextAuthorityLeaves = append(nextAuthorityLeaves, crypto.Keccak256(v))
			}

			nextAuthorityTree, err := merkle.NewTree(hasher.Keccak256Hasher{}).FromLeaves(nextAuthorityLeaves)
			require.NoError(t, err)

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
				t.Log("Initializing client state")
				continue
			}

			t.Logf("Recieved Commitment #%d", count)

			// first get all paraIds
			// fetch all registered parachainIds, this method doesn't account for
			// if the parachains whose header was included in the batch of finalized blocks have now
			// lost their parachain slot at this height
			paraIds, err := fetchParaIDs(relayApi, blockHash)
			require.NoError(t, err)

			var paraHeaderKeys []clientTypes.StorageKey

			// create full storage key for our own paraId
			keyPrefix := clientTypes.CreateStorageKeyPrefix("Paras", "Heads")
			// so we can query all blocks from lastfinalized to latestBeefyHeight
			encodedParaID, err := types.Encode(PARA_ID)
			require.NoError(t, err)

			twoXHash := xxhash.New64(encodedParaID).Sum(nil)
			// full key path in the storage source: https://www.shawntabrizi.com/assets/presentations/substrate-storage-deep-dive.pdf
			// xx128("Paras") + xx128("Heads") + xx64(Encode(paraId)) + Encode(paraId)
			fullKey := append(append(keyPrefix, twoXHash[:]...), encodedParaID...)
			paraHeaderKeys = append(paraHeaderKeys, fullKey)

			previousFinalizedHash, err := relayApi.RPC.Chain.GetBlockHash(uint64(clientState.LatestBeefyHeight + 1))
			require.NoError(t, err)

			changeSet, err := relayApi.RPC.State.QueryStorage(paraHeaderKeys, previousFinalizedHash, blockHash)
			require.NoError(t, err)

			// double map that holds block numbers, for which parachain header
			// was included in the mmr leaf, seeing as our parachain headers might not make it into
			// every relay chain block.
			// Map<BlockNumber, Map<ParaId, Header>>
			var finalizedBlocks = make(map[uint32]map[uint32][]byte)

			// request for batch mmr proof of those leaves
			var leafIndices []uint64

			for _, changes := range changeSet {
				header, err := relayApi.RPC.Chain.GetHeader(changes.Block)
				require.NoError(t, err)

				var heads = make(map[uint32][]byte)

				for _, paraId := range paraIds {
					header, err := fetchParachainHeader(relayApi, paraId, changes.Block)
					require.NoError(t, err)
					heads[paraId] = header
				}

				finalizedBlocks[uint32(header.Number)] = heads

				leafIndices = append(leafIndices, uint64(clientState.GetLeafIndexForBlockNumber(uint32(header.Number))))
			}

			// fetch mmr proofs for leaves containing our target paraId
			mmrBatchProof, err := relayApi.RPC.MMR.GenerateBatchProof(leafIndices, blockHash)
			require.NoError(t, err)

			var parachainHeaders []*types.ParachainHeader

			var paraHeads = make([][]byte, len(mmrBatchProof.Leaves))

			for i := 0; i < len(mmrBatchProof.Leaves); i++ {
				type LeafWithIndex struct {
					Leaf  clientTypes.MmrLeaf
					Index uint64
				}

				v := LeafWithIndex{Leaf: mmrBatchProof.Leaves[i], Index: uint64(mmrBatchProof.Proof.LeafIndex[i])}
				paraHeads[i] = v.Leaf.ParachainHeads[:]
				var leafBlockNumber = clientState.GetBlockNumberForLeaf(uint32(v.Index))
				paraHeaders := finalizedBlocks[leafBlockNumber]

				var paraHeadsLeaves [][]byte
				// index of our parachain header in the
				// parachain heads merkle root
				var index uint32

				count := 0

				// sort by paraId
				var sortedParaIds []uint32
				for paraId := range paraHeaders {
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

				tree, err := merkle.NewTree(hasher.Keccak256Hasher{}).FromLeaves(paraHeadsLeaves)
				require.NoError(t, err)

				paraHeadsProof := tree.Proof([]uint64{uint64(index)})
				authorityRoot := bytes32(v.Leaf.BeefyNextAuthoritySet.Root[:])
				parentHash := bytes32(v.Leaf.ParentNumberAndHash.Hash[:])

				header := types.ParachainHeader{
					ParachainHeader: paraHeaders[PARA_ID],
					MmrLeafPartial: &types.BeefyMmrLeafPartial{
						Version:      types.U8(v.Leaf.Version),
						ParentNumber: v.Leaf.ParentNumberAndHash.ParentNumber,
						ParentHash:   &parentHash,
						BeefyNextAuthoritySet: types.BeefyAuthoritySet{
							Id:            v.Leaf.BeefyNextAuthoritySet.ID,
							Len:           v.Leaf.BeefyNextAuthoritySet.Len,
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

			mmrProof, err := relayApi.RPC.MMR.GenerateProof(
				uint64(clientState.GetLeafIndexForBlockNumber(blockNumber)),
				blockHash,
			)
			require.NoError(t, err)

			latestLeaf := mmrProof.Leaf

			BeefyNextAuthoritySetRoot := bytes32(latestLeaf.BeefyNextAuthoritySet.Root[:])
			parentHash := bytes32(latestLeaf.ParentNumberAndHash.Hash[:])

			var latestLeafMmrProof = make([][]byte, len(mmrProof.Proof.Items))
			for i := 0; i < len(mmrProof.Proof.Items); i++ {
				latestLeafMmrProof[i] = mmrProof.Proof.Items[i][:]
			}
			var mmrBatchProofItems = make([][]byte, len(mmrBatchProof.Proof.Items))
			for i := 0; i < len(mmrBatchProof.Proof.Items); i++ {
				mmrBatchProofItems[i] = mmrBatchProof.Proof.Items[i][:]
			}
			var signatures []*types.CommitmentSignature
			var authorityIndices []uint64
			// luckily for us, this is already sorted and maps to the right authority index in the authority root.
			for i, v := range signedCommitment.Signatures {
				if v.IsSome() {
					_, sig := v.Unwrap()
					signatures = append(signatures, &types.CommitmentSignature{
						Signature:      sig[:],
						AuthorityIndex: uint32(i),
					})
					authorityIndices = append(authorityIndices, uint64(i))
				}
			}

			CommitmentPayload := signedCommitment.Commitment.Payload[0]
			var payloadId types.SizedByte2 = CommitmentPayload.Id
			ParachainHeads := bytes32(latestLeaf.ParachainHeads[:])
			leafIndex := clientState.GetLeafIndexForBlockNumber(blockNumber)

			mmrUpdateProof := types.MmrUpdateProof{
				MmrLeaf: &types.BeefyMmrLeaf{
					Version:        types.U8(latestLeaf.Version),
					ParentNumber:   latestLeaf.ParentNumberAndHash.ParentNumber,
					ParentHash:     &parentHash,
					ParachainHeads: &ParachainHeads,
					BeefyNextAuthoritySet: types.BeefyAuthoritySet{
						Id:            latestLeaf.BeefyNextAuthoritySet.ID,
						Len:           latestLeaf.BeefyNextAuthoritySet.Len,
						AuthorityRoot: &BeefyNextAuthoritySetRoot,
					},
				},
				MmrLeafIndex: uint64(leafIndex),
				MmrProof:     latestLeafMmrProof,
				SignedCommitment: &types.SignedCommitment{
					Commitment: &types.Commitment{
						Payload:        []*types.PayloadItem{{PayloadId: &payloadId, PayloadData: CommitmentPayload.Value}},
						BlockNumer:     uint32(signedCommitment.Commitment.BlockNumber),
						ValidatorSetId: uint64(signedCommitment.Commitment.ValidatorSetID),
					},
					Signatures: signatures,
				},
				AuthoritiesProof: authorityTree.Proof(authorityIndices).ProofHashes(),
			}

			header := types.Header{
				ParachainHeaders: parachainHeaders,
				MmrProofs:        mmrBatchProofItems,
				MmrSize:          mmr.LeafIndexToMMRSize(uint64(leafIndex)),
				MmrUpdateProof:   &mmrUpdateProof,
			}

			err = clientState.VerifyClientMessage(sdk.Context{}, nil, nil, &header)
			require.NoError(t, err)

			t.Logf("clientState.LatestBeefyHeight: %d clientState.MmrRootHash: %s", clientState.LatestBeefyHeight, hex.EncodeToString(clientState.MmrRootHash))

			if clientState.LatestBeefyHeight != uint32(signedCommitment.Commitment.BlockNumber) {
				require.Equal(t, clientState.MmrRootHash, signedCommitment.Commitment.Payload, "failed to update client state. LatestBeefyHeight: %d, Commitment.BlockNumber %d", clientState.LatestBeefyHeight, uint32(signedCommitment.Commitment.BlockNumber))
			}
			t.Log("====== successfully processed justification! ======")

			// if UPDATE_STATE_MODE == "true" {
			// 	paramSpace := types2.NewSubspace(nil, nil, nil, nil, "test")
			// 	//paramSpace = paramSpace.WithKeyTable(clientypes.ParamKeyTable())

			// 	k := keeper.NewKeeper(nil, nil, paramSpace, nil, nil)
			// 	ctx := sdk.Context{}
			// 	store := k.ClientStore(ctx, "1234")

			// 	clientState.UpdateState(sdk.Context{}, nil, store, &header)
			// }

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
		return nil, fmt.Errorf("beefy authorities not found")
	}

	// Convert from ecdsa public key to ethereum address
	var authorityEthereumAddresses [][]byte
	for _, authority := range authorities {
		pub, err := crypto.DecompressPubkey(authority[:])
		if err != nil {
			return nil, err
		}
		ethereumAddress := crypto.PubkeyToAddress(*pub)
		authorityEthereumAddresses = append(authorityEthereumAddresses, ethereumAddress[:])
	}

	return authorityEthereumAddresses, nil
}

func fetchParachainHeader(conn *client.SubstrateAPI, paraId uint32, blockHash clientTypes.Hash) ([]byte, error) {
	// Fetch metadata
	meta, err := conn.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	paraIdEncoded := make([]byte, 4)
	binary.LittleEndian.PutUint32(paraIdEncoded, paraId)

	storageKey, err := clientTypes.CreateStorageKey(meta, "Paras", "Heads", paraIdEncoded)

	if err != nil {
		return nil, err
	}

	var parachainHeaders []byte

	ok, err := conn.RPC.State.GetStorage(storageKey, &parachainHeaders, blockHash)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("parachain header not found")
	}

	return parachainHeaders, nil
}

func fetchParaIDs(conn *client.SubstrateAPI, blockHash clientTypes.Hash) ([]uint32, error) {
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
		return nil, fmt.Errorf("beefy authorities not found")
	}

	return paraIds, nil
}
