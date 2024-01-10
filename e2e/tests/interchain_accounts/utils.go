package interchainaccounts

import (
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
)

func DefaultTestMetadataVersionString(controllerConnectionID, hostConnectionID string) string {
	metadata := icatypes.NewMetadata(icatypes.Version, controllerConnectionID, hostConnectionID, "", icatypes.EncodingProtobuf, icatypes.TxTypeSDKMultiMsg)
	versionBytes, _ := icatypes.ModuleCdc.MarshalJSON(&metadata)

	return string(versionBytes)
}
