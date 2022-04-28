package types_test

import (
	"bytes"
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/ComposableFi/go-substrate-rpc-client/v4/scale"
	substrateTypes "github.com/ComposableFi/go-substrate-rpc-client/v4/types"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
)

func TestDecodeParachainHeader(t *testing.T) {
	headerBytes, err := hex.DecodeString("7edf044b273544342c4dc30a234c327405b3b03f2f20f53fc6a41d6d2765536d38efc4d9b628f9ddb17b542822e3df456b5431c62a005a67bb593d30da23f2e57581004e468f3616573199929694b06fc4248c449621f1e04b7c1dc3135bc1f6e9080642414245340200000000bdaca9200000000005424142450101b4061c25a6260134de85942c551d75d7e29e660a8b090a4ec08051b32dad7253e7536a1214d06648c865a44a10ffd7a457f8d62c5783b55fd29d0faa1912c885")
	require.NoError(t, err, "error decoding parachain bytes")

	var header substrateTypes.Header
	err = types.DecodeFromBytes(headerBytes, &header)
	require.NoError(t, err, "error decoding parachain header")

	parentHash, err := hex.DecodeString("7edf044b273544342c4dc30a234c327405b3b03f2f20f53fc6a41d6d2765536d")
	require.NoError(t, err)

	require.Equal(t, header.ParentHash[:], parentHash[:], "error comparing decoded parent hash")

	extrinsicsRoot, err := hex.DecodeString("81004e468f3616573199929694b06fc4248c449621f1e04b7c1dc3135bc1f6e9")
	require.NoError(t, err)

	require.Equal(t, header.ExtrinsicsRoot[:], extrinsicsRoot[:], "error comparing extrinsicsRoot")

	stateRoot, err := hex.DecodeString("efc4d9b628f9ddb17b542822e3df456b5431c62a005a67bb593d30da23f2e575")
	require.NoError(t, err)

	require.Equal(t, header.StateRoot[:], stateRoot[:], "error comparing StateRoot")

	require.Equal(t, 14, header.Number, "failed to check block number from decoded header")

}

func TestDecodeExtrinsicTimestamp(t *testing.T) {
	var timeUnix uint64 = 1643972151006
	timestampBytes, err := hex.DecodeString("280403000bde4660c47e01")
	require.NoError(t, err)

	var extrinsic substrateTypes.Extrinsic
	err = types.DecodeFromBytes(timestampBytes, &extrinsic)
	require.NoError(t, err)

	unix, err := scale.NewDecoder(bytes.NewReader(extrinsic.Method.Args[:])).DecodeUintCompact()
	require.NoError(t, err)

	require.Equal(t, timeUnix, unix.Uint64(), "failed to decode unix timestamp")

}

func TestHeader(t *testing.T) {
	h := substrateTypes.Header{
		Digest: substrateTypes.Digest{
			substrateTypes.DigestItem{
				AsConsensus: substrateTypes.Consensus{
					Bytes: substrateTypes.NewBytes([]byte("/IBC")),
				},
			},
		},
	}

	require.Equal(t, []byte("/IBC"), []byte("/IBC")[:])
	var buffer = bytes.Buffer{}

	encoderInstance := scale.NewEncoder(&buffer)
	err := encoderInstance.Encode(h.Digest[0].AsConsensus)
	require.NoError(t, err)

	require.Len(t, h.Digest[0].AsConsensus.Bytes, 4)
	require.Equal(t, substrateTypes.NewBytes([]byte("/IBC")), h.Digest[0].AsConsensus.Bytes)
}
