package types

import (
	"errors"
	"fmt"

	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

// NewPacketState creates a new PacketState instance.
func NewPacketState(channelID string, sequence uint64, data []byte) PacketState {
	return PacketState{
		ChannelId: channelID,
		Sequence:  sequence,
		Data:      data,
	}
}

// Validate performs basic validation of fields returning an error upon any failure.
func (ps PacketState) Validate() error {
	if ps.Data == nil {
		return errors.New("data bytes cannot be nil")
	}
	return validateGenFields(ps.ChannelId, ps.Sequence)
}

// NewPacketSequence creates a new PacketSequences instance.
func NewPacketSequence(channelID string, sequence uint64) PacketSequence {
	return PacketSequence{
		ChannelId: channelID,
		Sequence:  sequence,
	}
}

// Validate performs basic validation of fields returning an error upon any failure.
func (ps PacketSequence) Validate() error {
	return validateGenFields(ps.ChannelId, ps.Sequence)
}

// NewGenesisState creates a GenesisState instance.
func NewGenesisState(
	channels []IdentifiedChannel, acks, receipts, commitments []PacketState,
	sendSeqs []PacketSequence, nextChannelSequence uint64,
) GenesisState {
	return GenesisState{
		Channels:            channels,
		Acknowledgements:    acks,
		Receipts:            receipts,
		Commitments:         commitments,
		SendSequences:       sendSeqs,
		NextChannelSequence: nextChannelSequence,
	}
}

// DefaultGenesisState returns the ibc channel v2 submodule's default genesis state.
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Channels:            []IdentifiedChannel{},
		Acknowledgements:    []PacketState{},
		Receipts:            []PacketState{},
		Commitments:         []PacketState{},
		SendSequences:       []PacketSequence{},
		NextChannelSequence: 0,
	}
}

// Validate performs basic genesis state validation returning an error upon any failure.
func (gs GenesisState) Validate() error {
	// keep track of the max sequence to ensure it is less than
	// the next sequence used in creating channel identifiers.
	var maxSequence uint64

	for i, channel := range gs.Channels {
		sequence, err := channeltypesv1.ParseChannelSequence(channel.ChannelId)
		if err != nil {
			return err
		}

		if sequence > maxSequence {
			maxSequence = sequence
		}

		if err := channel.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid channel %v channel index %d: %w", channel, i, err)
		}
	}

	if maxSequence != 0 && maxSequence >= gs.NextChannelSequence {
		return fmt.Errorf("next channel sequence %d must be greater than maximum sequence used in channel identifier %d", gs.NextChannelSequence, maxSequence)
	}

	for i, ack := range gs.Acknowledgements {
		if err := ack.Validate(); err != nil {
			return fmt.Errorf("invalid acknowledgement %v ack index %d: %w", ack, i, err)
		}
		if len(ack.Data) == 0 {
			return fmt.Errorf("invalid acknowledgement %v ack index %d: data bytes cannot be empty", ack, i)
		}
	}

	for i, receipt := range gs.Receipts {
		if err := receipt.Validate(); err != nil {
			return fmt.Errorf("invalid acknowledgement %v ack index %d: %w", receipt, i, err)
		}
	}

	for i, commitment := range gs.Commitments {
		if err := commitment.Validate(); err != nil {
			return fmt.Errorf("invalid commitment %v index %d: %w", commitment, i, err)
		}
		if len(commitment.Data) == 0 {
			return fmt.Errorf("invalid acknowledgement %v ack index %d: data bytes cannot be empty", commitment, i)
		}
	}

	for i, ss := range gs.SendSequences {
		if err := ss.Validate(); err != nil {
			return fmt.Errorf("invalid send sequence %v index %d: %w", ss, i, err)
		}
	}

	return nil
}

func validateGenFields(channelID string, sequence uint64) error {
	if err := host.ChannelIdentifierValidator(channelID); err != nil {
		return fmt.Errorf("invalid channel Id: %w", err)
	}
	if sequence == 0 {
		return errors.New("sequence cannot be 0")
	}
	return nil
}
