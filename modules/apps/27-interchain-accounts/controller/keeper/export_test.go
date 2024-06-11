package keeper

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
)

// GetICS4Wrapper is a getter for the keeper's ICS4Wrapper.
func (k *Keeper) GetICS4Wrapper() porttypes.ICS4Wrapper {
	return k.ics4Wrapper
}

// GetAppMetadata is a wrapper around getAppMetadata to allow the function to be directly called in tests.
func (k Keeper) GetAppMetadata(ctx sdk.Context, portID, channelID string) (icatypes.Metadata, error) {
	return k.getAppMetadata(ctx, portID, channelID)
}
