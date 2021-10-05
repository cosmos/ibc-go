package utils

import (
	"github.com/davecgh/go-spew/spew"
	abci "github.com/tendermint/tendermint/abci/types"
)

func AccumulateChanges(currentChanges []abci.ValidatorUpdate, newChanges []abci.ValidatorUpdate) []abci.ValidatorUpdate {
	m := make(map[string]abci.ValidatorUpdate)

	for i := 0; i < len(currentChanges); i++ {
		m[currentChanges[i].PubKey.String()] = currentChanges[i]
	}

	for i := 0; i < len(newChanges); i++ {
		m[newChanges[i].PubKey.String()] = newChanges[i]
	}

	var out []abci.ValidatorUpdate

	for _, update := range m {
		out = append(out, update)
	}

	spew.Dump(out)
	return out
}
