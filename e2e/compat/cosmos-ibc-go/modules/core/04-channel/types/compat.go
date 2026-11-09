// SPDX-License-Identifier: Apache-2.0

// Package types re-exports github.com/lfdt-ibc/ibc-go/v11/modules/core/04-channel/types for dependencies that still use the
// pre-rename import path.
package types

import lfdt "github.com/lfdt-ibc/ibc-go/v11/modules/core/04-channel/types"

type (
	MsgAcknowledgement = lfdt.MsgAcknowledgement
	MsgTimeout         = lfdt.MsgTimeout
)

var ErrInvalidChannelOrdering = lfdt.ErrInvalidChannelOrdering
