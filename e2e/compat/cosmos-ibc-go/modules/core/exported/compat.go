// SPDX-License-Identifier: Apache-2.0

// Package exported re-exports github.com/lfdt-ibc/ibc-go/v11/modules/core/exported for dependencies that still use the
// pre-rename import path.
package exported

import lfdt "github.com/lfdt-ibc/ibc-go/v11/modules/core/exported"

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
