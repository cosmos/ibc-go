// SPDX-License-Identifier: Apache-2.0

// Package exported re-exports github.com/lfdt-ibc/ibc-go/v11/modules/core/exported for dependencies that still use the
// pre-rename import path.
package exported

import (
	// interchaintest ships its own copy of the 08-wasm types (chain/cosmos/08-wasm-types),
	// which imports this package and registers the same ibc.lightclients.wasm.v1 proto
	// names. gogoproto keeps whichever registration runs first and leaves the other
	// copy without a type URL. Importing the ibc-go types here initializes them first,
	// as before the rename, when github.com/cosmos/ibc-go sorted ahead of
	// github.com/cosmos/interchaintest in Go's package initialization order.
	_ "github.com/lfdt-ibc/ibc-go/modules/light-clients/08-wasm/v11/types"

	lfdt "github.com/lfdt-ibc/ibc-go/v11/modules/core/exported"
)

type (
	ClientMessage   = lfdt.ClientMessage
	ClientState     = lfdt.ClientState
	ConsensusState  = lfdt.ConsensusState
	GenesisMetadata = lfdt.GenesisMetadata
	Height          = lfdt.Height
	Path            = lfdt.Path
	Status          = lfdt.Status
)

const (
	Active                = lfdt.Active
	LocalhostConnectionID = lfdt.LocalhostConnectionID
)
