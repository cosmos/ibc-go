// SPDX-License-Identifier: Apache-2.0

// Package host re-exports github.com/lfdt-ibc/ibc-go/v11/modules/core/24-host for dependencies that still use the
// pre-rename import path.
package host

import lfdt "github.com/lfdt-ibc/ibc-go/v11/modules/core/24-host"

var (
	ChannelIdentifierValidator = lfdt.ChannelIdentifierValidator
	PortIdentifierValidator    = lfdt.PortIdentifierValidator
)
