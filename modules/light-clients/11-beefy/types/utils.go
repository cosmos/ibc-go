package types

import (
	"bytes"
	"github.com/centrifuge/go-substrate-rpc-client/scale"
)

// Decode decodes an encoded type to a target type. It takes encoded bytes and target interface as arguments and
// returns decoded data as the target type.
func Decode(source []byte, target interface{}) error {
	dec := scale.NewDecoder(bytes.NewReader(source))
	err := dec.Decode(target)
	if err != nil {
		return err
	}
	return nil
}

// Encode scale encodes a data type and returns the scale encoded data as a byte type.
func Encode(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := scale.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}