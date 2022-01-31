package types_test

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/ComposableFi/go-merkle-trees/merkle"
	"github.com/ComposableFi/go-merkle-trees/mmr"
	client "github.com/ComposableFi/go-substrate-rpc-client/v4"
	clientTypes "github.com/ComposableFi/go-substrate-rpc-client/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type logCommitment struct {
	blockNumber    clientTypes.U32                    `json:"block_number"`
	payload        string                             `json:"payload"`
	validatorSetID clientTypes.U64                    `json:"validator_set_id"`
	signatures     []clientTypes.OptionBeefySignature `json:"signatures"`
	rawMessage     string                             `json:"raw_message"`
}

func PrettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("Witnessed a new BEEFY commitment. %s \n", string(b))
	return nil
}

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

			signedCommitment := &clientTypes.SignedCommitment{}

			err := types.DecodeFromHexString(msg.(string), signedCommitment)
			if err != nil {
				panic(err.Error())
			}

			output := logCommitment{
				blockNumber:    signedCommitment.Commitment.BlockNumber,
				payload:        signedCommitment.Commitment.Payload.Hex(),
				validatorSetID: signedCommitment.Commitment.ValidatorSetID,
			}
			fmt.Printf("Witnessed a new BEEFY commitment. %+v \n", output)

			// fmt.Printf("Witnessed a new BEEFY commitment. %+v \n", logCommitment{
			// 	blockNumber:    signedCommitment.Commitment.BlockNumber,
			// 	payload:        signedCommitment.Commitment.Payload.Hex(),
			// 	validatorSetID: signedCommitment.Commitment.ValidatorSetID,
			// 	rawMessage:     msg.(string),
			// 	signatures:     signedCommitment.Signatures,
			// })

			blockNumber := uint64(signedCommitment.Commitment.BlockNumber)
			blockHash, err := relayApi.RPC.Chain.GetBlockHash(blockNumber)
			if err != nil {
				panic(err)
			}

			authorities, err := getBeefyAuthorities(blockNumber, relayApi, "Authorities")
			if err != nil {
				panic(err)
			}

			paraHeads, err := fetchParaHeads(relayApi, blockHash)
			if err != nil {
				panic("Failed to decode BEEFY commitment messages")
			}

			// Log paraHeads as hex strings
			for k, v := range paraHeads {
				fmt.Printf("key: %d, paraHead: %s\n", k, hex.EncodeToString(v))
			}

			nextAuthorities, err := getBeefyAuthorities(blockNumber, relayApi, "NextAuthorities")
			if err != nil {
				panic(err)
			}

			var authorityLeaves [][]byte
			for _, v := range authorities {
				authorityLeaves = append(authorityLeaves, crypto.Keccak256(v))
			}

			var nextAuthorityLeaves [][]byte
			for _, v := range authorities {
				nextAuthorityLeaves = append(nextAuthorityLeaves, crypto.Keccak256(v))
			}

			authorityTree, err := merkle.NewTree(types.Keccak256{}).FromLeaves(authorityLeaves)
			if err != nil {
				panic(err)
			}
			nextAuthorityTree, err := merkle.NewTree(types.Keccak256{}).FromLeaves(nextAuthorityLeaves)
			if err != nil {
				panic(err)
			}

			if clientState == nil {
				clientState = &types.ClientState{
					MmrRootHash:          signedCommitment.Commitment.Payload[:],
					LatestBeefyHeight:    blockNumber,
					BeefyActivationBlock: 0,
					Authority: &types.BeefyAuthoritySet{
						Id:            uint64(signedCommitment.Commitment.ValidatorSetID),
						Len:           uint32(len(authorities)),
						AuthorityRoot: authorityTree.Root(),
					},
					NextAuthoritySet: &types.BeefyAuthoritySet{
						Id:            uint64(signedCommitment.Commitment.ValidatorSetID) + 1,
						Len:           uint32(len(nextAuthorities)),
						AuthorityRoot: nextAuthorityTree.Root(),
					},
				}
				fmt.Printf("Initializing client state %v", clientState)
				continue
			}

			var paraHeadsLeaves [][]byte
			var index uint32
			var paraHeader []byte
			count := 0

			sortedParaHeadKeys := func() []uint32 {
				var keys []uint32
				for k, _ := range paraHeads {
					keys = append(keys, k)
				}
				sort.SliceStable(keys, func(i, j int) bool {
					return keys[i] < keys[j]
				})
				return keys
			}

			for _, v := range sortedParaHeadKeys() {
				paraIdScale := make([]byte, 4)
				// scale encode para_id
				binary.LittleEndian.PutUint32(paraIdScale[:], v)
				leaf := append(paraIdScale, paraHeads[v]...)
				paraHeadsLeaves = append(paraHeadsLeaves, crypto.Keccak256(leaf))
				if v == 2000 {
					paraHeader = paraHeads[v]
					index = uint32(count)
				}
				count++
			}

			tree, err := merkle.NewTree(types.Keccak256{}).FromLeaves(paraHeadsLeaves)
			if err != nil {
				panic(err)
			}

			// todo: convert block number to leafIndex
			mmrProofs, err := relayApi.RPC.MMR.GenerateProof(blockNumber-1, blockHash)
			if err != nil {
				panic(err)
			}

			paraHeadsProof := tree.Proof([]uint32{index})

			parachainHeader := []*types.ParachainHeader{{
				ParachainHeader: paraHeader,
				MmrLeafPartial: &types.BeefyMmrLeafPartial{
					Version:      uint8(mmrProofs.Leaf.Version),
					ParentNumber: uint64(mmrProofs.Leaf.ParentNumberAndHash.ParentNumber),
					ParentHash:   mmrProofs.Leaf.ParentNumberAndHash.Hash[:],
					BeefyNextAuthoritySet: types.BeefyAuthoritySet{
						Id:            uint64(mmrProofs.Leaf.BeefyNextAuthoritySet.ID),
						Len:           uint32(mmrProofs.Leaf.BeefyNextAuthoritySet.Len),
						AuthorityRoot: mmrProofs.Leaf.BeefyNextAuthoritySet.Root[:],
					},
				},
				ParachainHeadsProof: paraHeadsProof.ProofHashes(),
				ParaId:              2000,
				HeadsLeafIndex:      index,
				HeadsTotalCount:     uint32(len(paraHeadsLeaves)),
			}}

			var proofItems [][]byte
			for i := 0; i < len(mmrProofs.Proof.Items); i++ {
				proofItems = append(proofItems, mmrProofs.Proof.Items[i][:])
			}
			var signatures []*types.CommitmentSignature
			var authorityIndeces []uint32
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
			header := types.Header{
				ParachainHeaders: parachainHeader,
				MmrProofs:        proofItems,
				MmrSize:          mmr.LeafIndexToMMRSize(uint64(mmrProofs.Proof.LeafIndex)),
				MmrUpdateProof: &types.MmrUpdateProof{
					MmrLeaf: &types.BeefyMmrLeaf{
						Version:        uint8(mmrProofs.Leaf.Version),
						ParentNumber:   uint64(mmrProofs.Leaf.ParentNumberAndHash.ParentNumber),
						ParentHash:     mmrProofs.Leaf.ParentNumberAndHash.Hash[:],
						ParachainHeads: mmrProofs.Leaf.ParachainHeads[:],
					},
					MmrLeafIndex: uint64(mmrProofs.Proof.LeafIndex),
					MmrProof:     proofItems,
					SignedCommitment: &types.SignedCommitment{
						Commitment: &types.Commitment{
							Payload: signedCommitment.Commitment.Payload[:],
							// Payload:        []*types.PayloadItem{{PayloadId: []byte("mh"), PayloadData: signedCommitment.Commitment.Payload[:]}},
							BlockNumer:     uint64(signedCommitment.Commitment.BlockNumber),
							ValidatorSetId: uint64(signedCommitment.Commitment.ValidatorSetID),
						},
						Signatures: signatures,
					},
					AuthoritiesProof: authorityTree.Proof(authorityIndeces).ProofHashes(),
				},
			}

			_, _, errs := clientState.CheckHeaderAndUpdateState(sdk.Context{}, nil, nil, &header)
			if errs != nil {
				panic(errs)
			}
			fmt.Printf("====== successfully processed justification! ======\n")

		}
	}
}

type Authorities = [][33]uint8

func getBeefyAuthorities(blockNumber uint64, conn *client.SubstrateAPI, method string) ([][]byte, error) {
	blockHash, err := conn.RPC.Chain.GetBlockHash(blockNumber)
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

	// Convert from beefy authorities to ethereum addresses
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

func fetchParaHeads(conn *client.SubstrateAPI, blockHash clientTypes.Hash) (map[uint32][]byte, error) {

	keyPrefix := clientTypes.CreateStorageKeyPrefix("Paras", "Heads")

	keys, err := conn.RPC.State.GetKeys(keyPrefix, blockHash)
	if err != nil {
		fmt.Errorf("Failed to get all parachain keys %v \n", err)
		return nil, err
	}

	changeSets, err := conn.RPC.State.QueryStorageAt(keys, blockHash)
	if err != nil {
		fmt.Errorf("Failed to get all parachain headers %v \n", err)
		return nil, err
	}

	heads := make(map[uint32][]byte)

	for _, changeSet := range changeSets {
		for _, change := range changeSet.Changes {

			var paraID uint32

			if err := types.DecodeFromBytes(change.StorageKey[40:], &paraID); err != nil {
				fmt.Errorf("Failed to decode parachain ID %v \n", err)
				return nil, err
			}

			headDataWrapped := change.StorageData

			var headData clientTypes.Bytes
			if err := types.DecodeFromBytes(headDataWrapped, &headData); err != nil {
				fmt.Errorf("Failed to decode HeadData wrapper %v \n", err)
				return nil, err
			}

			heads[paraID] = headData
		}
	}

	return heads, nil
}
