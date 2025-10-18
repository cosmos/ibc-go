package blsverifier

/*
 * This custom query handler is used to aggregate public keys and verify a signature using BLS.
 * It is used by the 08-wasm union light client, which we we use in the solidity IBC v2 e2e tests.
 * The code here is taken from here: https://github.com/unionlabs/union/tree/main/uniond/app/custom_query
 */
import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MessageSize = 32
)

type CustomQuery struct {
	AggregateVerify *QueryAggregateVerify `json:"aggregate_verify,omitempty"`
	Aggregate       *QueryAggregate       `json:"aggregate,omitempty"`
}
type QueryAggregate struct {
	PublicKeys [][]byte `json:"public_keys"`
}
type QueryAggregateVerify struct {
	PublicKeys [][]byte `json:"public_keys"`
	Signature  []byte   `json:"signature"`
	Message    []byte   `json:"message"`
}

func CustomQuerier() func(sdk.Context, json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var customQuery CustomQuery
		err := json.Unmarshal([]byte(request), &customQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to parse custom query %w", err)
		}

		switch {
		case customQuery.Aggregate != nil:
			aggregatedPublicKeys, err := AggregatePublicKeys(customQuery.Aggregate.PublicKeys)
			if err != nil {
				return nil, fmt.Errorf("failed to aggregate public keys %w", err)
			}

			return json.Marshal(aggregatedPublicKeys.Marshal())
		case customQuery.AggregateVerify != nil:
			if len(customQuery.AggregateVerify.Message) != MessageSize {
				return nil, fmt.Errorf("invalid message length (%d), must be a %d bytes hash: %x", len(customQuery.AggregateVerify.Message), MessageSize, customQuery.AggregateVerify.Message)
			}

			msg := [MessageSize]byte{}
			for i := range MessageSize {
				msg[i] = customQuery.AggregateVerify.Message[i]
			}
			result, err := VerifySignature(customQuery.AggregateVerify.Signature, msg, customQuery.AggregateVerify.PublicKeys)
			if err != nil {
				return nil, fmt.Errorf("failed to verify signature %w", err)
			}

			return json.Marshal(result)
		default:
			return nil, fmt.Errorf("unknown custom query %v", request)
		}
	}
}
