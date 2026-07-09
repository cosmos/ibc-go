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

// ValidatePendingPacketParts validates the string fields used by pending packet
// collection keys. Pending packet keys are stored as (channelID, denom,
// sequence), where channelID and denom are non-terminal string keys and cannot
// contain the collections string delimiter byte 0x00.
func ValidatePendingPacketParts(channelID, denom string) error {
	if err := validatePendingPacketChannelID(channelID); err != nil {
		return err
	}

	if denom == "" {
		return errors.New("pending packet denom must be specified")
	}
	if strings.ContainsRune(denom, '\x00') {
		return errors.New("pending packet denom cannot contain 0x00")
	}

	return nil
}

func validatePendingPacketChannelID(channelID string) error {
	if len(channelID) > PendingSendPacketChannelLength {
		return errorsmod.Wrapf(ErrInvalidChannelID, "channel %s with length %d is greater than the allowed length %d", channelID, len(channelID), PendingSendPacketChannelLength)
	}
	if strings.ContainsRune(channelID, '\x00') {
		return errorsmod.Wrapf(ErrInvalidChannelID, "channel ID %q cannot contain 0x00", channelID)
	}

	return nil
}

// IsLegacyPendingPacketID returns true if the pending packet ID is in the pre-denom
// genesis format. These IDs cannot be safely migrated to denom-scoped markers and
// are dropped on import, mirroring the store migration behavior.
func IsLegacyPendingPacketID(pendingPacketID string) bool {
	splits := strings.Split(pendingPacketID, "/")
	if len(splits) != 2 {
		return false
	}

	_, err := strconv.ParseUint(splits[1], 10, 64)
	if err != nil {
		return false
	}

	return validatePendingPacketChannelID(splits[0]) == nil
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
	channelOrClientID, _, denom, err := ParsePendingPacketID(pendingPacketID)
	if err != nil {
		if IsLegacyPendingPacketID(pendingPacketID) {
			return nil
		}
		return err
	}

	return ValidatePendingPacketParts(channelOrClientID, denom)
}
