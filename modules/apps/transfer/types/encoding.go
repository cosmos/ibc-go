package types

import (
	"bytes"
	"encoding/json"
)

// UnmarshalJSON implements the Unmarshaller interface for FungibleTokenPacketData.
func (ftpd *FungibleTokenPacketData) UnmarshalJSON(bz []byte) error {
	// Recursion protection. We cannot unmarshal into FungibleTokenPacketData directly
	// else UnmarshalJSON is going to get invoked again, ad infinum. Create an alias
	// and unmarshal into that, instead.
	type ftpdAlias FungibleTokenPacketData

	d := json.NewDecoder(bytes.NewReader(bz))
	// Raise errors during decoding if unknown fields are encountered.
	d.DisallowUnknownFields()

	var alias ftpdAlias
	if err := d.Decode(&alias); err != nil {
		return err
	}

	*ftpd = FungibleTokenPacketData(alias)
	return nil
}
