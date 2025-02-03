package ibctesting

import (
	"encoding/hex"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/api/cometbft/abci/v1"

	"github.com/cosmos/gogoproto/proto"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// ParsePacketV2FromEvents parses events emitted from a send packet and returns
// the first EventTypeSendPacket packet found.
// Returns an error if no packet is found.
func ParsePacketV2FromEvents(events []abci.Event) (channeltypesv2.Packet, error) {
	packets, err := ParsePacketsV2FromEvents(channeltypesv2.EventTypeSendPacket, events)
	if err != nil {
		return channeltypesv2.Packet{}, err
	}
	return packets[0], nil
}

// ParsePacketsV2FromEvents parses events emitted from a MsgRecvPacket and returns
// all the packets found.
// Returns an error if no packet is found.
func ParsePacketsV2FromEvents(eventtype string, events []abci.Event) ([]channeltypesv2.Packet, error) {
	ferr := func(err error) ([]channeltypesv2.Packet, error) {
		return nil, fmt.Errorf("ibctesting.ParsePacketsFromEvents: %w", err)
	}
	var packets []channeltypesv2.Packet
	for _, ev := range events {
		if ev.Type == eventtype {
			var packet channeltypesv2.Packet
			for _, attr := range ev.Attributes {
				if attr.Key == channeltypesv2.AttributeKeyEncodedPacketHex {
					encodedPacket, err := hex.DecodeString(attr.Value)
					if err != nil {
						return ferr(err)
					}
					if err := proto.Unmarshal(encodedPacket, &packet); err != nil {
						return ferr(err)
					}
				}
			}
			packets = append(packets, packet)
		}
	}
	return packets, nil
}

func ParseAckV2FromEvents(events []abci.Event) (*channeltypesv2.Acknowledgement, error) {
	for _, ev := range events {
		if ev.Type == channeltypesv2.EventTypeWriteAck {
			if attribute, found := attributeByKey(ev.Attributes, channeltypesv2.AttributeKeyEncodedAckHex); found {
				value, err := hex.DecodeString(attribute.Value)
				if err != nil {
					return nil, err
				}
				var ack channeltypesv2.Acknowledgement
				if err := proto.Unmarshal(value, &ack); err != nil {
					return nil, err
				}
				return &ack, nil
			}
		}
	}
	return nil, errors.New("acknowledgement event attribute not found")

}
