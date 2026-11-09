// SPDX-License-Identifier: Apache-2.0

// Package types re-exports github.com/lfdt-ibc/ibc-go/v11/modules/apps/27-interchain-accounts/types for dependencies that still use the
// pre-rename import path.
package types

import lfdt "github.com/lfdt-ibc/ibc-go/v11/modules/apps/27-interchain-accounts/types"

type InterchainAccountPacketData = lfdt.InterchainAccountPacketData

const EXECUTE_TX = lfdt.EXECUTE_TX //nolint:revive // must match the name interchaintest uses

var SerializeCosmosTx = lfdt.SerializeCosmosTx
