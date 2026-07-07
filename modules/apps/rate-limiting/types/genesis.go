package types

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
)

// Splits a pending packet of the form {channelId}/{sequenceNumber}/{denom} into
// the channel ID, sequence number, and denom respectively.
func ParsePendingPacketID(pendingPacketID string) (string, uint64, string, error) {
	splits := strings.SplitN(pendingPacketID, "/", 3)
	if len(splits) != 3 {
		return "", 0, "", fmt.Errorf("invalid pending packet (%s), must be of form: {channelId}/{sequenceNumber}/{denom}", pendingPacketID)
	}
	channelID := splits[0]
	sequenceString := splits[1]
	denom := splits[2]
	if denom == "" {
		return "", 0, "", fmt.Errorf("invalid pending packet (%s), denom must be specified", pendingPacketID)
	}

	sequence, err := strconv.ParseUint(sequenceString, 10, 64)
	if err != nil {
		return "", 0, "", errorsmod.Wrapf(err, "unable to parse sequence number (%s) from pending packet, %s", sequenceString, err)
	}

	return channelID, sequence, denom, nil
}

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		RateLimits:                       []RateLimit{},
		WhitelistedAddressPairs:          []WhitelistedAddressPair{},
		BlacklistedDenoms:                make([]string, 0),
		PendingSendPacketSequenceNumbers: make([]string, 0),
		PendingRecvPacketSequenceNumbers: make([]string, 0),
		HourEpoch: HourEpoch{
			EpochNumber: 0,
			Duration:    time.Hour,
		},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	for _, pendingPacketID := range gs.PendingSendPacketSequenceNumbers {
		if err := validatePendingPacketID(pendingPacketID); err != nil {
			return err
		}
	}
	for _, pendingPacketID := range gs.PendingRecvPacketSequenceNumbers {
		if err := validatePendingPacketID(pendingPacketID); err != nil {
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

func validatePendingPacketID(pendingPacketID string) error {
	channelOrClientID, sequence, _, err := ParsePendingPacketID(pendingPacketID)
	if err != nil {
		return err
	}

	_, err = PendingPacketKey(channelOrClientID, sequence)
	return err
}
