package types_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/ComposableFi/go-substrate-rpc-client/v4/scale"
	substrateTypes "github.com/ComposableFi/go-substrate-rpc-client/v4/types"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
)

func TestDecodeParachainHeader(t *testing.T) {
	headerBytes, err := hex.DecodeString("7edf044b273544342c4dc30a234c327405b3b03f2f20f53fc6a41d6d2765536d38efc4d9b628f9ddb17b542822e3df456b5431c62a005a67bb593d30da23f2e57581004e468f3616573199929694b06fc4248c449621f1e04b7c1dc3135bc1f6e9080642414245340200000000bdaca9200000000005424142450101b4061c25a6260134de85942c551d75d7e29e660a8b090a4ec08051b32dad7253e7536a1214d06648c865a44a10ffd7a457f8d62c5783b55fd29d0faa1912c885")
	if err != nil {
		t.Errorf("error decoding parachain bytes: %s", err)
	}

	var header substrateTypes.Header
	scaleErr := types.DecodeFromBytes(headerBytes, &header)
	if scaleErr != nil {
		t.Errorf("error decoding parachain header: %s", err)
	}
	hash, err := substrateTypes.NewHashFromHexString("0x7edf044b273544342c4dc30a234c327405b3b03f2f20f53fc6a41d6d2765536d")
	if err != nil {
		panic(err)
	}
	if bytes.Equal(header.ParentHash[:], hash[:]) {
		t.Errorf("error decoding parent hash")
	}

	if header.Number != 14 {
		t.Errorf("check block number from decoded header:  got: %d, want %d", header.Number, 1)
	}
}

func TestDecodeExtrinsicTimestamp(t *testing.T) {
	var timeUnix uint64 = 1643972151006
	timestampBytes, err := hex.DecodeString("280403000bde4660c47e01")
	if err != nil {
		panic(err)
	}
	var extrinsic substrateTypes.Extrinsic
	decodeErr := types.DecodeFromBytes(timestampBytes, &extrinsic)
	if decodeErr != nil {
		panic(decodeErr)
	}
	unix, unixDecodeErr := scale.NewDecoder(bytes.NewReader(extrinsic.Method.Args[:])).DecodeUintCompact()
	if unixDecodeErr != nil {
		panic(unixDecodeErr)
	}

	if timeUnix != unix.Uint64() {
		t.Errorf("Failed to decode %d, %d", timeUnix, unix.Uint64())
	}

}
