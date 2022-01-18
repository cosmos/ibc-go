package types_test

import (
	"bytes"
	"github.com/centrifuge/go-substrate-rpc-client/scale"
	substrateTypes "github.com/centrifuge/go-substrate-rpc-client/types"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/11-beefy/types"
	"testing"
	"time"
)

func encode(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := scale.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestDecodeParachainHeader(t *testing.T) {
	hash, err := substrateTypes.NewHashFromHexString("da5e6d0616e05c6a6348605a37ca33493fc1a15ad1e6a405ee05c17843fdafed")
	if err != nil {
		panic(err)
	}

	data := substrateTypes.Header{
		ParentHash: hash,
		Number: 1,
	}

	encoded, err := encode(data)
	if err != nil {
		panic(err)
	}

	header, err := types.DecodeParachainHeader(encoded)
	if err != nil {
		t.Errorf("error decoding parachain header: %s", err)
	}

	if header.ParentHash != hash {
		t.Errorf("check parentHash from decoded header:  got: %s, want %s", header.ParentHash, hash)
	}

	if header.Number != 1 {
		t.Errorf("check block number from decoded header:  got: %d, want %d", header.Number, 1)
	}
}

func TestDecodeExtrinsicTimestamp(t *testing.T) {
	var timeUnix int64 = 1257894000
	encodedTimestamp, err := time.Unix(timeUnix, 0).GobEncode()
	if err != nil {
		panic(err)
	}

	data := substrateTypes.NewExtrinsic(substrateTypes.Call{Args: encodedTimestamp})
	buf := bytes.Buffer{}
	err = data.Encode(*scale.NewEncoder(&buf))
	if err != nil {
		panic(err)
	}

	decodedTimestamp, err := types.DecodeExtrinsicTimestamp(buf.Bytes())
	if err != nil {
		t.Errorf("error decoding extrinsic timestamp: %s", err)
	}

	if decodedTimestamp.Unix() != timeUnix {
		t.Errorf("check timestamp from decoded extrinsic:  got: %d, want %d", decodedTimestamp.UnixNano(), timeUnix)
	}
}