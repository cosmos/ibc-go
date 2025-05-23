package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
)

// Initializes a new flow from the channel value
func NewFlow(channelValue sdkmath.Int) Flow {
	flow := Flow{
		ChannelValue: channelValue,
		Inflow:       sdkmath.ZeroInt(),
		Outflow:      sdkmath.ZeroInt(),
	}

	return flow
}

// Adds an amount to the rate limit's flow after an incoming packet was received
// Returns an error if the new inflow will cause the rate limit to exceed its quota
func (f *Flow) AddInflow(amount sdkmath.Int, quota Quota) error {
	netInflow := f.Inflow.Sub(f.Outflow).Add(amount)

	if quota.CheckExceedsQuota(PACKET_RECV, netInflow, f.ChannelValue) {
		return errorsmod.Wrapf(ErrQuotaExceeded,
			"Inflow exceeds quota - Net Inflow: %v, Channel Value: %v, Threshold: %v%%",
			netInflow, f.ChannelValue, quota.MaxPercentRecv)
	}

	f.Inflow = f.Inflow.Add(amount)
	return nil
}

// Adds an amount to the rate limit's flow after a packet was sent
// Returns an error if the new outflow will cause the rate limit to exceed its quota
func (f *Flow) AddOutflow(amount sdkmath.Int, quota Quota) error {
	netOutflow := f.Outflow.Sub(f.Inflow).Add(amount)

	if quota.CheckExceedsQuota(PACKET_SEND, netOutflow, f.ChannelValue) {
		return errorsmod.Wrapf(ErrQuotaExceeded,
			"Outflow exceeds quota - Net Outflow: %v, Channel Value: %v, Threshold: %v%%",
			netOutflow, f.ChannelValue, quota.MaxPercentSend)
	}

	f.Outflow = f.Outflow.Add(amount)
	return nil
}
