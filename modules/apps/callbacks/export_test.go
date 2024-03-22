package ibccallbacks

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
)

// ProcessCallback is a wrapper around processCallback to allow the function to be directly called in tests.
func (im IBCMiddleware) ProcessCallback(
	ctx sdk.Context, callbackType types.CallbackType,
	callbackData types.CallbackData, callbackExecutor func(sdk.Context) error,
) error {
	return im.processCallback(ctx, callbackType, callbackData, callbackExecutor)
}
