package types

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	time "time"

	errorsmod "cosmossdk.io/errors"
)

// Splits a pending send packet of the form {channelId}/{sequenceNumber} into the channel Id
// and sequence number respectively
func ParsePendingPacketId(pendingPacketId string) (channelId string, sequence uint64, err error) {
	splits := strings.Split(pendingPacketId, "/")
	if len(splits) != 2 {
		return "", 0, fmt.Errorf("invalid pending send packet (%s), must be of form: {channelId}/{sequenceNumber}", pendingPacketId)
	}
	channelId = splits[0]
	sequenceString := splits[1]

	sequence, err = strconv.ParseUint(sequenceString, 10, 64)
	if err != nil {
		return "", 0, errorsmod.Wrapf(err, "unable to parse sequence number (%s) from pending send packet, %s", sequenceString, err)
	}

	return channelId, sequence, nil
}

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// Params:                           DefaultParams(),
		RateLimits:                       []RateLimit{},
		WhitelistedAddressPairs:          []WhitelistedAddressPair{},
		BlacklistedDenoms:                []string{},
		PendingSendPacketSequenceNumbers: []string{},
		HourEpoch: HourEpoch{
			EpochNumber: 0,
			Duration:    time.Hour,
		},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Validate the format of the pending send packets
	for _, pendingPacketId := range gs.PendingSendPacketSequenceNumbers {
		if _, _, err := ParsePendingPacketId(pendingPacketId); err != nil {
			return err
		}
	}

	// Verify the epoch hour duration is specified
	if gs.HourEpoch.Duration == 0 {
		return errors.New("hour epoch duration must be specified")
	}

	// If the hour epoch has been initialized already (epoch number != 0), validate and then use it
	if gs.HourEpoch.EpochNumber > 0 {
		if gs.HourEpoch.EpochStartTime.Equal(time.Time{}) {
			return errors.New("if hour epoch number is non-empty, epoch time must be initialized")
		}
		if gs.HourEpoch.EpochStartHeight == 0 {
			return errors.New("if hour epoch number is non-empty, epoch height must be initialized")
		}
	}

	return nil
}
