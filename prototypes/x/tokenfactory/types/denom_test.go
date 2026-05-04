package types_test

import (
	"testing"

	"github.com/cosmos/sandbox-ledger/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

const (
	testDenom    = "testtoken"
	invalidDenom = "not-so-alphanumeric"
)

func TestValidateTokenFactoryDenom(t *testing.T) {
	tests := []struct {
		name      string
		denom     string
		expectErr error
	}{
		{
			name:      "valid denom",
			denom:     testDenom,
			expectErr: nil,
		},
		{
			name:      "valid denom, max length",
			denom:     "aaaaaaaaaaaaaaaaaaaa",
			expectErr: nil,
		},
		{
			name:      "valid denom, min length",
			denom:     "a",
			expectErr: nil,
		},
		{
			name:      "valid denom, alphanumeric",
			denom:     "testtoken1234567890",
			expectErr: nil,
		},
		{
			name:      "invalid denom, too long",
			denom:     "testtoken123456789012345678901",
			expectErr: types.ErrInvalidDenom,
		},
		{
			name:      "invalid denom, too short",
			denom:     "",
			expectErr: types.ErrInvalidDenom,
		},
		{
			name:      "invalid denom, non alphanumeric",
			denom:     invalidDenom,
			expectErr: types.ErrInvalidDenom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateTokenFactoryDenom(tt.denom)
			require.ErrorIs(t, err, tt.expectErr)
		})
	}
}
