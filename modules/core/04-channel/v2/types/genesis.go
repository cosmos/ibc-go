package types

import (
	"errors"
	"fmt"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// NewPacketState creates a new PacketState instance.
func NewPacketState(clientID string, sequence uint64, data []byte) PacketState {
	return PacketState{
		ClientId: clientID,
		Sequence: sequence,
		Data:     data,
	}
}

// Validate performs basic validation of fields returning an error upon any failure.
func (ps PacketState) Validate() error {
	if ps.Data == nil {
		return errors.New("data bytes cannot be nil")
	}
	return validateGenFields(ps.ClientId, ps.Sequence)
}

// NewPacketSequence creates a new PacketSequences instance.
func NewPacketSequence(clientID string, sequence uint64) PacketSequence {
	return PacketSequence{
		ClientId: clientID,
		Sequence: sequence,
	}
}

// Validate performs basic validation of fields returning an error upon any failure.
func (ps PacketSequence) Validate() error {
	return validateGenFields(ps.ClientId, ps.Sequence)
}

// NewGenesisState creates a GenesisState instance.
func NewGenesisState(
	acks, receipts, commitments, asyncPackets []PacketState,
	sendSeqs []PacketSequence,
) GenesisState {
	return GenesisState{
		Acknowledgements: acks,
		Receipts:         receipts,
		Commitments:      commitments,
		AsyncPackets:     asyncPackets,
		SendSequences:    sendSeqs,
	}
}

// DefaultGenesisState returns the ibc channel v2 submodule's default genesis state.
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Acknowledgements: []PacketState{},
		Receipts:         []PacketState{},
		Commitments:      []PacketState{},
		AsyncPackets:     []PacketState{},
		SendSequences:    []PacketSequence{},
	}
}

// Validate performs basic genesis state validation returning an error upon any failure.
func (gs GenesisState) Validate() error {
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

	for i, ap := range gs.AsyncPackets {
		if err := ap.Validate(); err != nil {
			return fmt.Errorf("invalid async packet %v index %d: %w", ap, i, err)
		}
		if len(ap.Data) == 0 {
			return fmt.Errorf("invalid async packet %v index %d: data bytes cannot be empty", ap, i)
		}
	}

	for i, ss := range gs.SendSequences {
		if err := ss.Validate(); err != nil {
			return fmt.Errorf("invalid send sequence %v index %d: %w", ss, i, err)
		}
	}

	return nil
}

func validateGenFields(clientID string, sequence uint64) error {
	if err := host.ClientIdentifierValidator(clientID); err != nil {
		return fmt.Errorf("invalid channel Id: %w", err)
	}
	if sequence == 0 {
		return errors.New("sequence cannot be 0")
	}
	return nil
}
