// SPDX-License-Identifier: Apache-2.0

// Package types re-exports github.com/lfdt-ibc/ibc-go/v11/modules/core/05-port/types for dependencies that still use the
// pre-rename import path.
package types

import lfdt "github.com/lfdt-ibc/ibc-go/v11/modules/core/05-port/types"

var ErrInvalidPort = lfdt.ErrInvalidPort
