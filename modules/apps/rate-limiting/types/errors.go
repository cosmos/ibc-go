package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrRateLimitAlreadyExists = errorsmod.Register(ModuleName, 1, "ratelimit key duplicated")
	ErrRateLimitNotFound      = errorsmod.Register(ModuleName, 2, "rate limit not found")
	ErrZeroChannelValue       = errorsmod.Register(ModuleName, 3, "channel value is zero")
	ErrQuotaExceeded          = errorsmod.Register(ModuleName, 4, "quota exceeded")
	ErrInvalidClientState     = errorsmod.Register(ModuleName, 5, "unable to determine client state from channelId")
	ErrChannelNotFound        = errorsmod.Register(ModuleName, 6, "channel does not exist")
	ErrDenomIsBlacklisted     = errorsmod.Register(ModuleName, 7, "denom is blacklisted")
	ErrUnsupportedAttribute   = errorsmod.Register(ModuleName, 8, "unsupported attribute")
	ErrEpochNotFound          = errorsmod.Register(ModuleName, 9, "hour epoch not found in store")
	ErrUnmarshalEpoch         = errorsmod.Register(ModuleName, 10, "could not unmarshal epochBz")
	ErrInvalidEpoce           = errorsmod.Register(ModuleName, 11, "invalid hour epoch")
)
