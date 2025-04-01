
package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrRateLimitAlreadyExists = errorsmod.Register(ModuleName, 1,
		"ratelimit key duplicated")
	ErrRateLimitNotFound = errorsmod.Register(ModuleName, 2,
		"rate limit not found")
	ErrZeroChannelValue = errorsmod.Register(ModuleName, 3,
		"channel value is zero")
	ErrQuotaExceeded = errorsmod.Register(ModuleName, 4,
		"quota exceeded")
	ErrInvalidClientState = errorsmod.Register(ModuleName, 5,
		"unable to determine client state from channelId")
	ErrChannelNotFound = errorsmod.Register(ModuleName, 6,
		"channel does not exist")
	ErrDenomIsBlacklisted = errorsmod.Register(ModuleName, 7,
		"denom is blacklisted",
	)
)
