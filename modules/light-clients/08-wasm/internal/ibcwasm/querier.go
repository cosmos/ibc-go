package ibcwasm

import (
	"errors"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
)

type defaultQuerier struct{}

func (*defaultQuerier) GasConsumed() uint64 {
	return 0
}

func (*defaultQuerier) Query(_ wasmvmtypes.QueryRequest, _ uint64) ([]byte, error) {
	return nil, errors.New("queries in contract are not allowed")
}
