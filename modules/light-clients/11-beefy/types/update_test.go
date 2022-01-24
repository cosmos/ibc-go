package types_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
	store_test "github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	client "github.com/snowfork/go-substrate-rpc-client/v3"
	clientTypes "github.com/snowfork/go-substrate-rpc-client/v3/types"
	
)

type Authorities = [][33]uint8

func getBeefyAuthorities(blockNumber uint64, conn *client.SubstrateAPI, method string) ([]common.Address, error) {
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
	var authorityEthereumAddresses []common.Address
	for _, authority := range authorities {
		pub, err := crypto.DecompressPubkey(authority[:])
		if err != nil {
			return nil, err
		}
		ethereumAddress := crypto.PubkeyToAddress(*pub)
		if err != nil {
			return nil, err
		}
		authorityEthereumAddresses = append(authorityEthereumAddresses, ethereumAddress)
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
			if change.StorageData.IsNone() {
				continue
			}

			var paraID uint32

			if err := types.DecodeFromBytes(change.StorageKey[40:], &paraID); err != nil {
				fmt.Errorf("Failed to decode parachain ID %v \n", err)
				return nil, err
			}

			_, headDataWrapped := change.StorageData.Unwrap()

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
func TestCheckHeaderAndUpdateState(t *testing.T) {
	relayApi, err := client.NewSubstrateAPI("wss://127.0.0.1:9944")
	if err != nil {
		panic(err)
	}

	_parachainApi, err := client.NewSubstrateAPI("wss://127.0.0.1:9988")
	if err != nil {
		panic(err)
	}

	ch := make(chan interface{})

	sub, err := relayApi.Client.Subscribe(
		context.Background(), // todo:
		"beefy",
		"subscribeJustifications",
		"unsubscribeJustifications",
		"justifications",
		ch,
	)
	if err != nil {
		panic(err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				panic("error reading channel")
			}

			signedCommitment := &store_test.SignedCommitment{}
			err := types.DecodeFromHexString(msg.(string), signedCommitment)
			if err != nil {
				panic("Failed to decode BEEFY commitment messages")
			}

			fmt.Printf("Witnessed a new BEEFY commitment. %v \n", map[string]interface{}{
				"signedCommitment.Commitment.BlockNumber":    signedCommitment.Commitment.BlockNumber,
				"signedCommitment.Commitment.Payload":        signedCommitment.Commitment.Payload.Hex(),
				"signedCommitment.Commitment.ValidatorSetID": signedCommitment.Commitment.ValidatorSetID,
				"signedCommitment.Signatures":                signedCommitment.Signatures,
				"rawMessage":                                 msg.(string),
			})

			parentNumber := signedCommitment.Commitment.BlockNumber - 1
			blockHash, err := relayApi.RPC.Chain.GetBlockHash(uint64(signedCommitment.Commitment.BlockNumber))
			if err != nil {
				panic(err)
			}
			parentHash, err := relayApi.RPC.Chain.GetBlockHash(uint64(parentNumber))
			if err != nil {
				panic(err)
			}

			authorities, err := getBeefyAuthorities(uint64(signedCommitment.Commitment.BlockNumber), relayApi, "Authorities")
			if err != nil {
				panic(err)
			}

			paraHeads, err := fetchParaHeads(relayApi, blockHash)
			if err != nil {
				panic("Failed to decode BEEFY commitment messages")
			}
			nextAuthorities, err := getBeefyAuthorities(uint64(signedCommitment.Commitment.BlockNumber), relayApi, "NextAuthorities")
			if err != nil {
				panic(err)
			}
		}
	}

}
