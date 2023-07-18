package ibccallbacks

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// ProcessCallback is a wrapper around processCallback to allow the function to be directly called in tests.
func (im IBCMiddleware) ProcessCallback(
	ctx sdk.Context, packet channeltypes.Packet, callbackType types.CallbackType,
	callbackDataGetter func() (types.CallbackData, bool, error),
	callbackExecutor func(sdk.Context, string) error,
) error {
	return im.processCallback(ctx, packet, callbackType, callbackDataGetter, callbackExecutor)
}
