package types_test

import (
	"testing"

	"github.com/cosmos/sandbox-ledger/x/ift/types"
	"github.com/stretchr/testify/require"
)

func TestValidateCounterpartyAddress(t *testing.T) {
	tests := []struct {
		name            string
		constructorType string
		address         string
		expectErr       bool
	}{
		{
			name:            "EVM - valid address",
			constructorType: types.ConstructorEVM,
			address:         "0x742d35Cc6634C0532925a3b844Bc9e7595f3aD12",
			expectErr:       false,
		},
		{
			name:            "EVM - invalid address",
			constructorType: types.ConstructorEVM,
			address:         "invalid",
			expectErr:       true,
		},
		{
			name:            "Cosmos - valid address",
			constructorType: types.ConstructorCosmos,
			address:         "cosmos1abc123",
			expectErr:       false,
		},
		{
			name:            "Cosmos - empty address",
			constructorType: types.ConstructorCosmos,
			address:         "",
			expectErr:       true,
		},
		{
			name:            "Solana - valid address",
			constructorType: types.ConstructorSolana,
			address:         "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
			expectErr:       false,
		},
		{
			name:            "Solana - invalid address",
			constructorType: types.ConstructorSolana,
			address:         "invalid",
			expectErr:       true,
		},
		{
			name:            "unknown constructor type",
			constructorType: "unknown",
			address:         "any",
			expectErr:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := types.ValidateCounterpartyAddress(tc.constructorType, tc.address)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEVMConstructor_ValidateCounterpartyAddress(t *testing.T) {
	constructor := &types.EVMConstructor{}

	tests := []struct {
		name      string
		address   string
		expectErr bool
	}{
		{
			name:      "valid checksummed address",
			address:   "0x742d35Cc6634C0532925a3b844Bc9e7595f3aD12",
			expectErr: false,
		},
		{
			name:      "valid lowercase address",
			address:   "0x742d35cc6634c0532925a3b844bc9e7595f3ad12",
			expectErr: false,
		},
		{
			name:      "invalid - cosmos address",
			address:   "cosmos1abc123",
			expectErr: true,
		},
		{
			name:      "invalid - empty",
			address:   "",
			expectErr: true,
		},
		{
			name:      "invalid - too short",
			address:   "0x742d35",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := constructor.ValidateCounterpartyAddress(tc.address)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCosmosTxConstructor_ValidateCounterpartyAddress(t *testing.T) {
	constructor := &types.CosmosTxConstructor{}

	tests := []struct {
		name      string
		address   string
		expectErr bool
	}{
		{
			name:      "non-empty address",
			address:   "cosmos1abc123",
			expectErr: false,
		},
		{
			name:      "module account style",
			address:   "cosmos1ift",
			expectErr: false,
		},
		{
			name:      "invalid - empty",
			address:   "",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := constructor.ValidateCounterpartyAddress(tc.address)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidConstructorTypes(t *testing.T) {
	constructorTypes := types.ValidConstructorTypes()
	require.Len(t, constructorTypes, 3)
	require.Contains(t, constructorTypes, "evm")
	require.Contains(t, constructorTypes, "cosmos")
	require.Contains(t, constructorTypes, "solana")
}

func TestParseSolanaConfig(t *testing.T) {
	tests := []struct {
		name           string
		constructorStr string
		expectErr      bool
		errContains    string
		expectedCfg    *types.SolanaOptions
	}{
		{
			name:           "valid config",
			constructorStr: `{"solana":{"gmp_program_id":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","mint_pubkey":"11111111111111111111111111111111"}}`,
			expectErr:      false,
			expectedCfg: &types.SolanaOptions{
				GMPProgramID: "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
				MintPubKey:   "11111111111111111111111111111111",
			},
		},
		{
			name:           "not solana constructor - evm",
			constructorStr: "evm",
			expectErr:      true,
			errContains:    "invalid JSON",
		},
		{
			name:           "not solana constructor - cosmos",
			constructorStr: "cosmos",
			expectErr:      true,
			errContains:    "invalid JSON",
		},
		{
			name:           "invalid JSON",
			constructorStr: `{"solana":{invalid}}`,
			expectErr:      true,
			errContains:    "invalid JSON",
		},
		{
			name:           "empty solana config",
			constructorStr: `{"solana":{}}`,
			expectErr:      false,
			expectedCfg:    &types.SolanaOptions{},
		},
		{
			name:           "wrong key in JSON",
			constructorStr: `{"evm":{}}`,
			expectErr:      true,
			errContains:    "not a solana constructor",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := types.ParseSolanaConfig(tc.constructorStr)
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedCfg, cfg)
			}
		})
	}
}

func TestValidateSolanaConstructorString(t *testing.T) {
	tests := []struct {
		name           string
		constructorStr string
		expectErr      bool
		errContains    string
	}{
		{
			name:           "valid config",
			constructorStr: `{"solana":{"gmp_program_id":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","mint_pubkey":"11111111111111111111111111111111"}}`,
			expectErr:      false,
		},
		{
			name:           "invalid gmp_program_id",
			constructorStr: `{"solana":{"gmp_program_id":"invalid","mint_pubkey":"11111111111111111111111111111111"}}`,
			expectErr:      true,
			errContains:    "invalid gmp_program_id",
		},
		{
			name:           "invalid mint_pubkey",
			constructorStr: `{"solana":{"gmp_program_id":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","mint_pubkey":"invalid"}}`,
			expectErr:      true,
			errContains:    "invalid mint_pubkey",
		},
		{
			name:           "empty gmp_program_id",
			constructorStr: `{"solana":{"gmp_program_id":"","mint_pubkey":"11111111111111111111111111111111"}}`,
			expectErr:      true,
			errContains:    "invalid gmp_program_id",
		},
		{
			name:           "empty mint_pubkey",
			constructorStr: `{"solana":{"gmp_program_id":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","mint_pubkey":""}}`,
			expectErr:      true,
			errContains:    "invalid mint_pubkey",
		},
		{
			name:           "not solana constructor - plain string",
			constructorStr: "evm",
			expectErr:      true,
			errContains:    "invalid JSON",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := types.ValidateSolanaConstructorString(tc.constructorStr)
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
				require.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
			}
		})
	}
}

func TestValidateConstructorString(t *testing.T) {
	tests := []struct {
		name           string
		constructorStr string
		expectErr      bool
		errContains    string
	}{
		{
			name:           "valid EVM constructor",
			constructorStr: types.ConstructorEVM,
			expectErr:      false,
		},
		{
			name:           "valid Cosmos constructor",
			constructorStr: types.ConstructorCosmos,
			expectErr:      false,
		},
		{
			name:           "valid Solana constructor",
			constructorStr: `{"solana":{"gmp_program_id":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","mint_pubkey":"11111111111111111111111111111111"}}`,
			expectErr:      false,
		},
		{
			name:           "invalid Solana constructor - invalid gmp_program_id",
			constructorStr: `{"solana":{"gmp_program_id":"invalid","mint_pubkey":"11111111111111111111111111111111"}}`,
			expectErr:      true,
			errContains:    "invalid gmp_program_id",
		},
		{
			name:           "unknown constructor type",
			constructorStr: "unknown",
			expectErr:      true,
			errContains:    "unknown: unknown",
		},
		{
			name:           "empty constructor",
			constructorStr: "",
			expectErr:      true,
			errContains:    "unknown:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := types.ValidateConstructorString(tc.constructorStr)
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseConstructorType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"evm", "evm"},
		{"cosmos", "cosmos"},
		{`{"solana":{"key":"value"}}`, "solana"},
		{`{"evm":{}}`, "evm"},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := types.ParseConstructorType(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestSolanaConstructor_ValidateCounterpartyAddress(t *testing.T) {
	constructor := &types.SolanaConstructor{}

	tests := []struct {
		name      string
		address   string
		expectErr bool
	}{
		{
			name:      "valid solana address",
			address:   "11111111111111111111111111111111",
			expectErr: false,
		},
		{
			name:      "valid solana address - token program",
			address:   "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
			expectErr: false,
		},
		{
			name:      "valid solana address - random",
			address:   "9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin",
			expectErr: false,
		},
		{
			name:      "invalid - cosmos address",
			address:   "cosmos1abc123",
			expectErr: true,
		},
		{
			name:      "invalid - EVM address",
			address:   "0x742d35Cc6634C0532925a3b844Bc9e7595f3aD12",
			expectErr: true,
		},
		{
			name:      "invalid - empty",
			address:   "",
			expectErr: true,
		},
		{
			name:      "invalid - too short",
			address:   "abc123",
			expectErr: true,
		},
		{
			name:      "invalid - contains invalid base58 chars",
			address:   "0OIl11111111111111111111111111111",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := constructor.ValidateCounterpartyAddress(tc.address)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
