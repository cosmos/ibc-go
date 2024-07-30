package mock

import errorsmod "cosmossdk.io/errors"

var ErrInvalidClientMsg = errorsmod.Register(ModuleName, 1, "invalid client message")
