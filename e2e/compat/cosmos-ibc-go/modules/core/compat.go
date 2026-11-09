// SPDX-License-Identifier: Apache-2.0

// Package ibc re-exports github.com/lfdt-ibc/ibc-go/v11/modules/core for dependencies that still use the
// pre-rename import path.
package ibc

import lfdt "github.com/lfdt-ibc/ibc-go/v11/modules/core"

type AppModuleBasic = lfdt.AppModuleBasic
