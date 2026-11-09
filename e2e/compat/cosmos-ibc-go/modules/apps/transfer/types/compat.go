// SPDX-License-Identifier: Apache-2.0

// Package types re-exports github.com/lfdt-ibc/ibc-go/v11/modules/apps/transfer/types for dependencies that still use the
// pre-rename import path.
package types

import lfdt "github.com/lfdt-ibc/ibc-go/v11/modules/apps/transfer/types"

const PortID = lfdt.PortID

var (
	NewDenom = lfdt.NewDenom
	NewHop   = lfdt.NewHop
)
