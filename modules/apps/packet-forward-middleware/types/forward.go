package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// PacketMetadata represents the metadata for a packet in the packet-forward middleware.
// Do not use this type directly with json encoding/decoding, as it is not json serializable.
// Instead use the provided helper methods to convert it, or use the Metadata keys defined in this package.
type PacketMetadata struct {
	Forward ForwardMetadata
}

// ForwardMetadata represents the metadata for forwarding a packet.
// Do not use this type directly with json encoding/decoding, as it is not json serializable.
// Instead use the provided helper methods to convert it, or use the Metadata keys defined in this package.
type ForwardMetadata struct {
	Receiver string
	Port     string
	Channel  string
	Timeout  time.Duration
	Retries  *uint8

	Next *PacketMetadata // Next is a pointer to allow nil values
}

func (m ForwardMetadata) Validate() error {
	if m.Receiver == "" {
		return errors.New("failed to validate metadata. receiver cannot be empty")
	}
	if err := host.PortIdentifierValidator(m.Port); err != nil {
		return fmt.Errorf("failed to validate metadata: %w", err)
	}
	if err := host.ChannelIdentifierValidator(m.Channel); err != nil {
		return fmt.Errorf("failed to validate metadata: %w", err)
	}

	return nil
}

func (m ForwardMetadata) ToMap() map[string]any {
	forwardMetadataMap := map[string]any{
		ForwardReceiverKey: m.Receiver,
		ForwardPortKey:     m.Port,
		ForwardChannelKey:  m.Channel,
	}

	if m.Timeout > 0 {
		forwardMetadataMap[ForwardTimeoutKey] = m.Timeout
	}

	if m.Retries != nil {
		forwardMetadataMap[ForwardRetriesKey] = *m.Retries
	}

	if m.Next != nil {
		forwardMetadataMap[ForwardNextKey] = m.Next.toMap()
	}

	return forwardMetadataMap
}

func (m PacketMetadata) toMap() map[string]any {
	packetMetadataMap := map[string]any{
		ForwardMetadataKey: m.Forward.ToMap(),
	}

	return packetMetadataMap
}

func (m PacketMetadata) ToMemo() (string, error) {
	packetMetadataMap := map[string]any{
		ForwardMetadataKey: m.Forward.ToMap(),
	}

	packetMetadataJSON, err := json.Marshal(packetMetadataMap)
	if err != nil {
		return "", err
	}

	return string(packetMetadataJSON), nil
}

func GetPacketMetadataFromPacketdata(transferDetail ibcexported.PacketDataProvider) (PacketMetadata, bool, error) {
	forwardData, ok := transferDetail.GetCustomPacketData(ForwardMetadataKey).(map[string]any)
	if forwardData == nil || !ok {
		return PacketMetadata{}, false, errorsmod.Wrapf(ErrMetadataKeyNotFound, "key %s not found in packet data", ForwardMetadataKey)
	}

	forwardMetadata, err := getForwardMetadata(forwardData)
	if err != nil {
		return PacketMetadata{}, true, errorsmod.Wrapf(err, "failed to get forward metadata from packet data")
	}

	return PacketMetadata{
		Forward: forwardMetadata,
	}, true, nil
}

func getForwardMetadata(forwardData map[string]any) (ForwardMetadata, error) {
	receiver, ok := forwardData[ForwardReceiverKey].(string)
	if !ok {
		return ForwardMetadata{}, errorsmod.Wrapf(ErrMetadataKeyNotFound, "key %s not found in packet data", ForwardReceiverKey)
	}

	port, ok := forwardData[ForwardPortKey].(string)
	if !ok {
		return ForwardMetadata{}, errorsmod.Wrapf(ErrMetadataKeyNotFound, "key %s not found in packet data", ForwardPortKey)
	}

	channel, ok := forwardData[ForwardChannelKey].(string)
	if !ok {
		return ForwardMetadata{}, errorsmod.Wrapf(ErrMetadataKeyNotFound, "key %s not found in packet data", ForwardChannelKey)
	}

	var err error
	timeout := time.Duration(0)
	timeoutData, ok := forwardData[ForwardTimeoutKey]
	if ok {
		timeout, err = parseDuration(timeoutData)
		if err != nil {
			return ForwardMetadata{}, err
		}
	}

	var retries *uint8
	retriesData, ok := forwardData[ForwardRetriesKey]
	if ok {
		retriesFloat, ok := retriesData.(float64)
		if !ok {
			return ForwardMetadata{}, errorsmod.Wrapf(ErrInvalidForwardMetadata, "key %s has invalid type, expected number", ForwardRetriesKey)
		}
		if retriesFloat < 0 || retriesFloat > 255 {
			return ForwardMetadata{}, errors.New("retries must be between 0 and 255")
		}
		retriesU8 := uint8(retriesFloat)
		retries = &retriesU8
	}

	var next *PacketMetadata
	nextDataAny, ok := forwardData[ForwardNextKey]
	if ok {
		nextData, err := getForwardMetadataFromNext(nextDataAny)
		if err != nil {
			return ForwardMetadata{}, errorsmod.Wrapf(err, "failed to get next data")
		}

		nextForward, err := getForwardMetadata(nextData)
		if err != nil {
			return ForwardMetadata{}, errorsmod.Wrapf(err, "failed to get next forward metadata from packet data")
		}

		next = &PacketMetadata{
			Forward: nextForward,
		}
	}

	return ForwardMetadata{
		Receiver: receiver,
		Port:     port,
		Channel:  channel,
		Timeout:  timeout,
		Retries:  retries,
		Next:     next,
	}, nil
}

func getForwardMetadataFromNext(nextData any) (map[string]any, error) {
	var packetMetadataMap map[string]any
	packetMetadataMap, ok := nextData.(map[string]any)
	if !ok {
		nextDataStr, ok := nextData.(string)
		if !ok {
			return nil, errorsmod.Wrapf(ErrInvalidForwardMetadata, "next forward metadata is not a valid map or string")
		}

		if err := json.Unmarshal([]byte(nextDataStr), &packetMetadataMap); err != nil {
			return nil, errorsmod.Wrapf(ErrInvalidForwardMetadata, "failed to unmarshal next forward metadata: %s", err.Error())
		}
	}

	forwardData, ok := packetMetadataMap[ForwardMetadataKey].(map[string]any)
	if !ok {
		return nil, errorsmod.Wrapf(ErrMetadataKeyNotFound, "key %s not found in next forward metadata", ForwardMetadataKey)
	}

	return forwardData, nil
}

func parseDuration(duration any) (time.Duration, error) {
	switch value := duration.(type) {
	case float64:
		return time.Duration(value), nil
	case string:
		return time.ParseDuration(value)
	default:
		return 0, errors.New("invalid duration")
	}
}
