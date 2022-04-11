package types_test

import (
	"encoding/hex"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ComposableFi/go-merkle-trees/mmr"
	substrate "github.com/ComposableFi/go-substrate-rpc-client/v4/types"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTrieProof(t *testing.T) {
	//TODO: enable it once #2329 is fixed
	t.Skip("NOT YET FIXED")

	key, err := hex.DecodeString("f0c365c3cf59d671eb72da0e7a4113c4bbd108c4899964f707fdaffb82636065")
	require.NoError(t, err)

	root, err := hex.DecodeString("5e1eb8e577ea88deaa94b456da24ab0c9f4c0c6d9372af1568edd7aeef3b4c4e")
	require.NoError(t, err)

	bytes1, err := hex.DecodeString("80fffd8028b54b9a0a90d41b7941c43e6a0597d5914e3b62bdcb244851b9fc806c28ea2480e2f0847174b6f8ea15133a8d70de58d1a6174b7542e8e12028154c611bc3ee5280ddd81bdda149a8bc6990d3548a719d4a90ddbe5ea4a598211aacf6afd0b23bf58038fe7e08c8e684bd600f25631f32e6510ed7d37f43fce0d5aa974009857aeb5b80aafc60caa3519d4b861e6b8da226266a15060e2071bba4184e194da61dfb208e80b34a4ee6e2f949f58b7cb7f4a7fb1aaea8cdc2a5cb27557d32da7096fdf157c58024a760a8f6c27928ae9e2fed9968bc5f6e17c3ae647398d8a615e5b2bb4b425f8085a0da830399f25fca4b653de654ffd3c92be39f3ae4f54e7c504961b5bd00cf80c2d44d371e5fc1f50227d7491ad65ad049630361cefb4ab1844831237609f08380c8ae6a1e8df858b43e050a3959a25b90d711413ee1a863622c3914d45250738980b5955ff982ab818fcba39b2d507a6723504cef4969fc7c722ee175df95a33ae280509bb016f2887d12137e73d26d7ddcd7f9c8ff458147cb9d309494655fe68de180009f8697d760fbe020564b07f407e6aad58ba9451b3d2d88b3ee03e12db7c47480952dcc0804e1120508a1753f1de4aa5b7481026a3320df8b48e918f0cecbaed380fff4f175da5ff30200fabfdc2bbdd45f864d84f339ec2432f80b5749ac35bbfc")
	require.NoError(t, err)

	bytes2, err := hex.DecodeString("9ec365c3cf59d671eb72da0e7a4113c41002505f0e7b9012096b41c4eb3aaf947f6ea429080000685f0f1f0515f462cdcf84e0f1d6045dfcbb2056145f077f010000")
	require.NoError(t, err)

	bytes3, err := hex.DecodeString("80050880149156720805d0ad098ae52fcffae34ff637b1d1f1a0fa8e7f94201b8615695580c1638f702aaa71e4b78cc8538ecae03e827bb494cc54279606b201ec071a5e24806d2a1e6d5236e1e13c5a5c84831f5f5383f97eba32df6f9faf80e32cf2f129bc")
	require.NoError(t, err)

	var proof = [][]byte{
		bytes1, bytes2, bytes3,
	}

	emptyTrie := trie.NewEmptyTrie()

	err = emptyTrie.LoadFromProof(proof, root)
	require.NoError(t, err)

	value := emptyTrie.Get(key)
	require.NotEmpty(t, value)
}

func TestMultiLeafMmrProofs(t *testing.T) {
	var opaqueLeaves []substrate.OpaqueLeafWithIndex
	err := types.DecodeFromHexString(
		"10c50100bd020000dbd670705fddee2d22d0d3cdced8734aa8c8374197eaf1493f9fb86e7fbeba0f010000000000000005000000baa93c7834125ee3120bac6e3342bd3f28611110ad21ab6075367abdffefeb0975e8469015638c96e3d9942cb35297b28d0cca7add9932c5a7354fa302d6f2e3bd02000000000000c50100be020000a76a43d5b7bf9bfa6c7562ebd35019f7709910a3f76464688f62717b13b20fe8010000000000000005000000baa93c7834125ee3120bac6e3342bd3f28611110ad21ab6075367abdffefeb0975e8469015638c96e3d9942cb35297b28d0cca7add9932c5a7354fa302d6f2e3be02000000000000c50100bf0200006d0ab0459bf2a9305e048805fbc9bc58b7e9454906c2e94b6d981f0e8ceb3180010000000000000005000000baa93c7834125ee3120bac6e3342bd3f28611110ad21ab6075367abdffefeb091b8751c2c1962bde4b57548c1bdef9ef2efcf93730e130036feee3f944bec9edbf02000000000000c50100c0020000d6aaaaa38e330ac1500a22d0fe382ffe2ca9f95161f479dca2e56038696ee343010000000000000005000000baa93c7834125ee3120bac6e3342bd3f28611110ad21ab6075367abdffefeb091b8751c2c1962bde4b57548c1bdef9ef2efcf93730e130036feee3f944bec9edc002000000000000",
		&opaqueLeaves,
	)
	require.NoError(t, err)

	var leaves []mmr.Leaf

	for _, leaf := range opaqueLeaves {
		leaves = append(leaves, mmr.Leaf{
			Index: leaf.Index,
			Hash:  crypto.Keccak256(leaf.Leaf),
		})
	}

	var batchProof substrate.MmrBatchProof

	err = types.DecodeFromHexString(
		"10bd02000000000000be02000000000000bf02000000000000c002000000000000e903000000000000380b69447305465f8796365fe6035c938e8307482a7eb81d312c74e3bdd4d06e6f861ff8ff2a2c35ba80caf31bbb1d5042133a61b8371af548477d7cf2fc7456ba2c831a65e8ca11b67a84f4b36a9cacb86a27b30e0cc0f10b7a4d406bbcf331e881ae35265781aa57e7619352caad12c681d6c07157f337f5b57a52491475289823ebb41b1af8e1213ae3159bd422d8b421d2813435d89c2dde3b3e201940a49eb282a3bda4a8cf9bef677ae1b49dc211cf25473e02fbf4aca9257552d91bb9763dca3cf547d4d15d53e4c9ee730e3acc8b3705359cbc2857eceea31121ed6706a0c991631e945495269afa5b3759915e77b62add69c1849ac742917e62922819b5c14bffd531d4ff99ef95b9f2e897d64e0e027439334d63cdb7d3ec0c988fa1aed09fb5b47b41a2e27946eecead11062188fb0353b813c1e74c23943a0497f9a5a92a54b9292f657ce45b9bcc699d4eac12a587f19878c51bb338c3c9d84f4481e964f7f7480b0ab9da1e691359b03c003cb3c2f5dc4a29ba9610167f0d782caad6b08a2ec0e74a66afea72837b5e070d18c5e79f0c1fc35fc9c5f0645811bcdaf53cb132d461ea60f4fe62f5a3fc1aa723f4854c067d84a3b1e26c93398bf9",
		&batchProof,
	)
	require.NoError(t, err)

	size := mmr.LeafIndexToMMRSize(uint64(batchProof.LeafCount - 1))

	var proofItems = make([][]byte, len(batchProof.Items))
	for i := range batchProof.Items {
		proofItems[i] = batchProof.Items[i][:]
	}

	expectedRoot := []byte{
		72, 183, 40, 135, 139, 221, 74, 166, 201, 0, 52, 167, 117, 108, 17, 181, 114, 52, 217,
		146, 200, 40, 236, 116, 241, 209, 1, 223, 30, 128, 62, 112,
	}

	root, err := mmr.NewProof(size, proofItems, leaves, types.Keccak256{}).CalculateRoot()
	require.NoError(t, err)

	require.Equal(t, expectedRoot, root)
}
