// SPDX-License-Identifier: Apache-2.0

// Package tendermint re-exports github.com/lfdt-ibc/ibc-go/v11/modules/light-clients/07-tendermint for dependencies that still use the
// pre-rename import path.
package tendermint

import lfdt "github.com/lfdt-ibc/ibc-go/v11/modules/light-clients/07-tendermint"

type AppModuleBasic = lfdt.AppModuleBasic
