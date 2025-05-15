package ibctesting

import (
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/cosmos/gogoproto/proto"
	testifysuite "github.com/stretchr/testify/suite"

	abci "github.com/cometbft/cometbft/abci/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

// ParseClientIDFromEvents parses events emitted from a MsgCreateClient and returns the
// client identifier.
func ParseClientIDFromEvents(events []abci.Event) (string, error) {
	for _, ev := range events {
		if ev.Type == clienttypes.EventTypeCreateClient {
			if attribute, found := attributeByKey(ev.Attributes, clienttypes.AttributeKeyClientID); found {
				return attribute.Value, nil
			}
		}
	}
	return "", errors.New("client identifier event attribute not found")
}

// ParseConnectionIDFromEvents parses events emitted from a MsgConnectionOpenInit or
// MsgConnectionOpenTry and returns the connection identifier.
func ParseConnectionIDFromEvents(events []abci.Event) (string, error) {
	for _, ev := range events {
		if ev.Type == connectiontypes.EventTypeConnectionOpenInit ||
			ev.Type == connectiontypes.EventTypeConnectionOpenTry {
			if attribute, found := attributeByKey(ev.Attributes, connectiontypes.AttributeKeyConnectionID); found {
				return attribute.Value, nil
			}
		}
	}
	return "", errors.New("connection identifier event attribute not found")
}

// ParseChannelIDFromEvents parses events emitted from a MsgChannelOpenInit or
// MsgChannelOpenTry or a MsgCreateChannel and returns the channel identifier.
func ParseChannelIDFromEvents(events []abci.Event) (string, error) {
	for _, ev := range events {
		if ev.Type == channeltypes.EventTypeChannelOpenInit || ev.Type == channeltypes.EventTypeChannelOpenTry {
			if attribute, found := attributeByKey(ev.Attributes, channeltypes.AttributeKeyChannelID); found {
				return attribute.Value, nil
			}
		}
	}
	return "", errors.New("channel identifier event attribute not found")
}

// ParseV1PacketFromEvents parses events emitted from a send packet and returns
// the first EventTypeSendPacket packet found.
// Returns an error if no packet is found.
func ParseV1PacketFromEvents(events []abci.Event) (channeltypes.Packet, error) {
	packets, err := ParseIBCV1Packets(channeltypes.EventTypeSendPacket, events)
	if err != nil {
		return channeltypes.Packet{}, err
	}
	return packets[0], nil
}

// ParseRecvV1PacketFromEvents parses events emitted from a MsgRecvPacket and returns
// the first EventTypeRecvPacket packet found.
// Returns an error if no packet is found.
func ParseRecvV1PacketFromEvents(events []abci.Event) (channeltypes.Packet, error) {
	packets, err := ParseIBCV1Packets(channeltypes.EventTypeRecvPacket, events)
	if err != nil {
		return channeltypes.Packet{}, err
	}
	return packets[0], nil
}

// ParseIBCV1Packets parses events and returns all the v1 packets found.
// Returns an error if no v1 packet is found.
func ParseIBCV1Packets(eventType string, events []abci.Event) ([]channeltypes.Packet, error) {
	ferr := func(err error) ([]channeltypes.Packet, error) {
		return nil, fmt.Errorf("ibctesting.ParseIBCV1Packets: %w", err)
	}
	var packets []channeltypes.Packet
	for _, ev := range events {
		if ev.Type != eventType {
			continue
		}

		var packet channeltypes.Packet
		for _, attr := range ev.Attributes {
			switch attr.Key {
			case channeltypes.AttributeKeyDataHex:
				data, err := hex.DecodeString(attr.Value)
				if err != nil {
					return ferr(err)
				}
				packet.Data = data
			case channeltypes.AttributeKeySequence:
				seq, err := strconv.ParseUint(attr.Value, 10, 64)
				if err != nil {
					return ferr(err)
				}

				packet.Sequence = seq

			case channeltypes.AttributeKeySrcPort:
				packet.SourcePort = attr.Value

			case channeltypes.AttributeKeySrcChannel:
				packet.SourceChannel = attr.Value

			case channeltypes.AttributeKeyDstPort:
				packet.DestinationPort = attr.Value

			case channeltypes.AttributeKeyDstChannel:
				packet.DestinationChannel = attr.Value

			case channeltypes.AttributeKeyTimeoutHeight:
				height, err := clienttypes.ParseHeight(attr.Value)
				if err != nil {
					return ferr(err)
				}

				packet.TimeoutHeight = height

			case channeltypes.AttributeKeyTimeoutTimestamp:
				timestamp, err := strconv.ParseUint(attr.Value, 10, 64)
				if err != nil {
					return ferr(err)
				}

				packet.TimeoutTimestamp = timestamp

			default:
				continue
			}
		}

		packets = append(packets, packet)
	}
	if len(packets) == 0 {
		return ferr(errors.New("acknowledgement event attribute not found"))
	}
	return packets, nil
}

// ParseIBCV2Packets parses events and returns all the v2 packets found.
// Returns an error if no v2 packet is found.
func ParseIBCV2Packets(eventType string, events []abci.Event) ([]channeltypesv2.Packet, error) {
	packets := make([]channeltypesv2.Packet, 0)
	for _, event := range events {
		if event.Type != eventType {
			continue
		}

		var packet channeltypesv2.Packet
	Loop:
		for _, attr := range event.Attributes {
			switch attr.Key {
			// If we find a complete packet, we unmarshall it. We don't need to check for any other
			// attributes from this event.
			case channeltypesv2.AttributeKeyEncodedPacketHex:
				data, err := hex.DecodeString(attr.Value)
				if err != nil {
					return nil, fmt.Errorf("ibctesting.ParseIBCV2Packets: %w", err)
				}
				if err := proto.Unmarshal(data, &packet); err != nil {
					return nil, fmt.Errorf("ibctesting.ParseIBCV2Packets: %w", err)
				}
				break Loop

			case channeltypesv2.AttributeKeySequence:
				seq, err := strconv.ParseUint(attr.Value, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("ibctesting.ParseIBCV2Packets: %w", err)
				}
				packet.Sequence = seq

			case channeltypesv2.AttributeKeyTimeoutTimestamp:
				timestamp, err := strconv.ParseUint(attr.Value, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("ibctesting.ParseIBCV2Packets: %w", err)
				}
				packet.TimeoutTimestamp = timestamp

			case channeltypesv2.AttributeKeyDstClient:
				packet.DestinationClient = attr.Value

			case channeltypesv2.AttributeKeySrcClient:
				packet.SourceClient = attr.Value
			}
		}
		packets = append(packets, packet)
	}

	if len(packets) == 0 {
		return nil, errors.New("no IBC v2 packets found in events")
	}

	return packets, nil
}

// ParseAckFromEvents parses events emitted from a MsgRecvPacket and returns the
// acknowledgement.
func ParseAckFromEvents(events []abci.Event) ([]byte, error) {
	for _, ev := range events {
		if ev.Type == channeltypes.EventTypeWriteAck {
			if attribute, found := attributeByKey(ev.Attributes, channeltypes.AttributeKeyAckHex); found {
				value, err := hex.DecodeString(attribute.Value)
				if err != nil {
					return nil, err
				}
				return value, nil
			}
		}
	}
	return nil, errors.New("acknowledgement event attribute not found")
}

// ParseProposalIDFromEvents parses events emitted from MsgSubmitProposal and returns proposalID
func ParseProposalIDFromEvents(events []abci.Event) (uint64, error) {
	for _, event := range events {
		if attribute, found := attributeByKey(event.Attributes, "proposal_id"); found {
			return strconv.ParseUint(attribute.Value, 10, 64)
		}
	}
	return 0, errors.New("proposalID event attribute not found")
}

// ParsePacketSequenceFromEvents parses events emitted from MsgRecvPacket and returns the packet sequence
func ParsePacketSequenceFromEvents(events []abci.Event) (uint64, error) {
	for _, event := range events {
		if attribute, found := attributeByKey(event.Attributes, "packet_sequence"); found {
			return strconv.ParseUint(attribute.Value, 10, 64)
		}
	}
	return 0, errors.New("packet sequence event attribute not found")
}

// AssertEvents asserts that expected events are present in the actual events.
func AssertEvents(
	suite *testifysuite.Suite,
	expected []abci.Event,
	actual []abci.Event,
) {
	foundEvents := make(map[int]bool)

	for i, expectedEvent := range expected {
		for _, actualEvent := range actual {
			if shouldProcessEvent(expectedEvent, actualEvent) {
				attributeMatch := true
				for _, expectedAttr := range expectedEvent.Attributes {
					// any expected attributes that are not contained in the actual events will cause this event
					// not to match
					attributeMatch = attributeMatch && containsAttribute(actualEvent.Attributes, expectedAttr.Key, expectedAttr.Value)
				}

				if attributeMatch {
					foundEvents[i] = true
				}
			}
		}
	}

	for i, expectedEvent := range expected {
		suite.Require().True(foundEvents[i], "event: %s was not found in events", expectedEvent.Type)
	}
}

// shouldProcessEvent returns true if the given expected event should be processed based on event type.
func shouldProcessEvent(expectedEvent abci.Event, actualEvent abci.Event) bool {
	if expectedEvent.Type != actualEvent.Type {
		return false
	}
	// the actual event will have an extra attribute added automatically
	// by Cosmos SDK since v0.50, that's why we subtract 1 when comparing
	// with the number of attributes in the expected event.
	if containsAttributeKey(actualEvent.Attributes, "msg_index") {
		return len(expectedEvent.Attributes) == len(actualEvent.Attributes)-1
	}

	return len(expectedEvent.Attributes) == len(actualEvent.Attributes)
}

// containsAttribute returns true if the given key/value pair is contained in the given attributes.
// NOTE: this ignores the indexed field, which can be set or unset depending on how the events are retrieved.
func containsAttribute(attrs []abci.EventAttribute, key, value string) bool {
	return slices.ContainsFunc(attrs, func(attr abci.EventAttribute) bool {
		return attr.Key == key && attr.Value == value
	})
}

// containsAttributeKey returns true if the given key is contained in the given attributes.
func containsAttributeKey(attrs []abci.EventAttribute, key string) bool {
	_, found := attributeByKey(attrs, key)
	return found
}

// attributeByKey returns the event attribute's value keyed by the given key and a boolean indicating its presence in the given attributes.
func attributeByKey(attributes []abci.EventAttribute, key string) (abci.EventAttribute, bool) {
	idx := slices.IndexFunc(attributes, func(a abci.EventAttribute) bool { return a.Key == key })
	if idx == -1 {
		return abci.EventAttribute{}, false
	}
	return attributes[idx], true
}

// ParsePacketFromEvents parses events emitted from a send packet and returns
// the first EventTypeSendPacket packet found.
// Returns an error if no packet is found.
//
// Deprecated: This function will be removed in the next major release. Use
// ParseV1PacketFromEvents instead
func ParsePacketFromEvents(events []abci.Event) (channeltypes.Packet, error) {
	return ParseV1PacketFromEvents(events)
}

// ParseRecvPacketFromEvents parses events emitted from a MsgRecvPacket and returns
// the first EventTypeRecvPacket packet found.
// Returns an error if no packet is found.
//
// Deprecated: This function will be removed in the next major release. Use
// ParseRecvV1PacketFromEvents instead
func ParseRecvPacketFromEvents(events []abci.Event) (channeltypes.Packet, error) {
	return ParseRecvV1PacketFromEvents(events)
}

// ParsePacketsFromEvents parses events emitted from a MsgRecvPacket and returns
// all the packets found.
// Returns an error if no packet is found.
//
// Deprecated: This function will be removed in the next major release. Use ParseIBCV1Packets instead.
func ParsePacketsFromEvents(eventType string, events []abci.Event) ([]channeltypes.Packet, error) {
	return ParseIBCV1Packets(eventType, events)
}
