package api

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"

	ibcerrors "github.com/cosmos/ibc-go/api/errors"
)

// ZeroHeight is a helper function which returns an uninitialized height.
func ZeroHeight() Height {
	return Height{}
}

// NewHeight is a constructor for the IBC height type
func NewHeight(revisionNumber, revisionHeight uint64) Height {
	return Height{
		RevisionNumber: revisionNumber,
		RevisionHeight: revisionHeight,
	}
}

// GetRevisionNumber returns the revision-number of the height
func (h Height) GetRevisionNumber() uint64 {
	return h.RevisionNumber
}

// GetRevisionHeight returns the revision-height of the height
func (h Height) GetRevisionHeight() uint64 {
	return h.RevisionHeight
}

// Compare implements a method to compare two heights. When comparing two heights a, b
// we can call a.Compare(b) which will return
// -1 if a < b
// 0  if a = b
// 1  if a > b
//
// It first compares based on revision numbers, whichever has the higher revision number is the higher height
// If revision number is the same, then the revision height is compared
func (h Height) Compare(other Height) int64 {
	var a, b big.Int
	if h.RevisionNumber != other.RevisionNumber {
		a.SetUint64(h.RevisionNumber)
		b.SetUint64(other.RevisionNumber)
	} else {
		a.SetUint64(h.RevisionHeight)
		b.SetUint64(other.RevisionHeight)
	}
	return int64(a.Cmp(&b))
}

// LT Helper comparison function returns true if h < other
func (h Height) LT(other Height) bool {
	return h.Compare(other) == -1
}

// LTE Helper comparison function returns true if h <= other
func (h Height) LTE(other Height) bool {
	cmp := h.Compare(other)
	return cmp <= 0
}

// GT Helper comparison function returns true if h > other
func (h Height) GT(other Height) bool {
	return h.Compare(other) == 1
}

// GTE Helper comparison function returns true if h >= other
func (h Height) GTE(other Height) bool {
	cmp := h.Compare(other)
	return cmp >= 0
}

// EQ Helper comparison function returns true if h == other
func (h Height) EQ(other Height) bool {
	return h.Compare(other) == 0
}

// String returns a string representation of Height
func (h Height) String() string {
	return fmt.Sprintf("%d-%d", h.RevisionNumber, h.RevisionHeight)
}

// Decrement will return a new height with the RevisionHeight decremented
// If the RevisionHeight is already at lowest value (1), then false success flag is returned
func (h Height) Decrement() (decremented Height, success bool) {
	if h.RevisionHeight == 0 {
		return Height{}, false
	}
	return NewHeight(h.RevisionNumber, h.RevisionHeight-1), true
}

// Increment will return a height with the same revision number but an
// incremented revision height
func (h Height) Increment() Height {
	return NewHeight(h.RevisionNumber, h.RevisionHeight+1)
}

// IsZero returns true if height revision and revision-height are both 0
func (h Height) IsZero() bool {
	return h.RevisionNumber == 0 && h.RevisionHeight == 0
}

// MustParseHeight will attempt to parse a string representation of a height and panic if
// parsing fails.
func MustParseHeight(heightStr string) Height {
	height, err := ParseHeight(heightStr)
	if err != nil {
		panic(err)
	}

	return height
}

// ParseHeight is a utility function that takes a string representation of the height
// and returns a Height struct
func ParseHeight(heightStr string) (Height, error) {
	splitStr := strings.Split(heightStr, "-")
	if len(splitStr) != 2 {
		return Height{}, errorsmod.Wrapf(ibcerrors.ErrInvalidHeight, "expected height string format: {revision}-{height}. Got: %s", heightStr)
	}
	revisionNumber, err := strconv.ParseUint(splitStr[0], 10, 64)
	if err != nil {
		return Height{}, errorsmod.Wrapf(ibcerrors.ErrInvalidHeight, "invalid revision number. parse err: %s", err)
	}
	revisionHeight, err := strconv.ParseUint(splitStr[1], 10, 64)
	if err != nil {
		return Height{}, errorsmod.Wrapf(ibcerrors.ErrInvalidHeight, "invalid revision height. parse err: %s", err)
	}
	return NewHeight(revisionNumber, revisionHeight), nil
}
