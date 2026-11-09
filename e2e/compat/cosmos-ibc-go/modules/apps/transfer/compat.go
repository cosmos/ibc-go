// SPDX-License-Identifier: Apache-2.0

// Package transfer re-exports github.com/lfdt-ibc/ibc-go/v11/modules/apps/transfer for dependencies that still use the
// pre-rename import path.
package transfer

import lfdt "github.com/lfdt-ibc/ibc-go/v11/modules/apps/transfer"

type AppModuleBasic = lfdt.AppModuleBasic
