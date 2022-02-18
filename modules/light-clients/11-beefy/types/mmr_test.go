package types_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ComposableFi/go-merkle-trees/mmr"
	substrate "github.com/ComposableFi/go-substrate-rpc-client/v4/types"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestMultiLeafMmrProofs(t *testing.T) {
	var opaqueLeaves []substrate.OpaqueLeafWithIndex
	err := types.DecodeFromHexString(
		"10c50100bd020000dbd670705fddee2d22d0d3cdced8734aa8c8374197eaf1493f9fb86e7fbeba0f010000000000000005000000baa93c7834125ee3120bac6e3342bd3f28611110ad21ab6075367abdffefeb0975e8469015638c96e3d9942cb35297b28d0cca7add9932c5a7354fa302d6f2e3bd02000000000000c50100be020000a76a43d5b7bf9bfa6c7562ebd35019f7709910a3f76464688f62717b13b20fe8010000000000000005000000baa93c7834125ee3120bac6e3342bd3f28611110ad21ab6075367abdffefeb0975e8469015638c96e3d9942cb35297b28d0cca7add9932c5a7354fa302d6f2e3be02000000000000c50100bf0200006d0ab0459bf2a9305e048805fbc9bc58b7e9454906c2e94b6d981f0e8ceb3180010000000000000005000000baa93c7834125ee3120bac6e3342bd3f28611110ad21ab6075367abdffefeb091b8751c2c1962bde4b57548c1bdef9ef2efcf93730e130036feee3f944bec9edbf02000000000000c50100c0020000d6aaaaa38e330ac1500a22d0fe382ffe2ca9f95161f479dca2e56038696ee343010000000000000005000000baa93c7834125ee3120bac6e3342bd3f28611110ad21ab6075367abdffefeb091b8751c2c1962bde4b57548c1bdef9ef2efcf93730e130036feee3f944bec9edc002000000000000",
		&opaqueLeaves,
	)
	if err != nil {
		panic(err)
	}
	var leaves []mmr.Leaf

	for _, leaf := range opaqueLeaves {
		leaves = append(leaves, mmr.Leaf{
			Index: leaf.Index,
			Hash: crypto.Keccak256(leaf.Leaf),
		})
	}

	var batchProof substrate.MmrBatchProof

	derr := types.DecodeFromHexString(
		"10bd02000000000000be02000000000000bf02000000000000c002000000000000e903000000000000380b69447305465f8796365fe6035c938e8307482a7eb81d312c74e3bdd4d06e6f861ff8ff2a2c35ba80caf31bbb1d5042133a61b8371af548477d7cf2fc7456ba2c831a65e8ca11b67a84f4b36a9cacb86a27b30e0cc0f10b7a4d406bbcf331e881ae35265781aa57e7619352caad12c681d6c07157f337f5b57a52491475289823ebb41b1af8e1213ae3159bd422d8b421d2813435d89c2dde3b3e201940a49eb282a3bda4a8cf9bef677ae1b49dc211cf25473e02fbf4aca9257552d91bb9763dca3cf547d4d15d53e4c9ee730e3acc8b3705359cbc2857eceea31121ed6706a0c991631e945495269afa5b3759915e77b62add69c1849ac742917e62922819b5c14bffd531d4ff99ef95b9f2e897d64e0e027439334d63cdb7d3ec0c988fa1aed09fb5b47b41a2e27946eecead11062188fb0353b813c1e74c23943a0497f9a5a92a54b9292f657ce45b9bcc699d4eac12a587f19878c51bb338c3c9d84f4481e964f7f7480b0ab9da1e691359b03c003cb3c2f5dc4a29ba9610167f0d782caad6b08a2ec0e74a66afea72837b5e070d18c5e79f0c1fc35fc9c5f0645811bcdaf53cb132d461ea60f4fe62f5a3fc1aa723f4854c067d84a3b1e26c93398bf9",
		&batchProof,
	)

	if derr != nil {
		panic(derr)
	}

	size := mmr.LeafIndexToMMRSize(uint64(batchProof.LeafCount - 1))

	var proofItems [][]byte
	for _, hash := range batchProof.Items {
		proofItems = append(proofItems, hash[:])
	}

	expectedRoot := []byte{
		72, 183, 40, 135, 139, 221, 74, 166, 201, 0, 52, 167, 117, 108, 17, 181, 114, 52, 217,
		146, 200, 40, 236, 116, 241, 209, 1, 223, 30, 128, 62, 112,
	}

	root, cerr := mmr.NewProof(size, proofItems, leaves, types.Keccak256{}).CalculateRoot()
	if cerr != nil {
		panic(cerr)
	}

	bytes.Equal(expectedRoot, root)

	fmt.Printf("Are they equal?: %t\n\n", bytes.Equal(expectedRoot, root))

}
