package types

import (
	"bytes"
	"encoding/json"
)

// UnmarshalJSON implements the Unmarshaller interface for FungibleTokenPacketData.
func (ack *IncentivizedAcknowledgement) UnmarshalJSON(bz []byte) error {
	// Recursion protection. We cannot unmarshal into IncentivizedAcknowledgment directly
	// else UnmarshalJSON is going to get invoked again, ad infinum. Create an alias
	// and unmarshal into that, instead.
	type ackAlias IncentivizedAcknowledgement

	d := json.NewDecoder(bytes.NewReader(bz))
	// Raise errors during decoding if unknown fields are encountered.
	d.DisallowUnknownFields()

	var alias ackAlias
	if err := d.Decode(&alias); err != nil {
		return err
	}

	*ack = IncentivizedAcknowledgement(alias)
	return nil
}
