package types

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// DefaultAllowedClients are the default clients for the AllowedClients parameter.
var (
	DefaultAllowedClients = []string{exported.Solomachine, exported.Tendermint, exported.Localhost}
	AllowAllClient        = "*"
)

// NewParams creates a new parameter configuration for the ibc client module
func NewParams(allowedClients ...string) Params {
	return Params{
		AllowedClients: allowedClients,
	}
}

// DefaultParams is the default parameter configuration for the ibc-client module.
func DefaultParams() Params {
	return NewParams(DefaultAllowedClients...)
}

func NewAllowAllClientParams() Params {
	return Params{
		AllowedClients: []string{AllowAllClient},
	}
}

// Validate all ibc-client module parameters
func (p Params) Validate() error {
	return validateClients(p.AllowedClients)
}

// IsAllowedClient checks if the given client type is registered on the allowlist.
func (p Params) IsAllowedClient(clientType string) bool {
	// Check for wildcard allow all client
	// If exist then allow all type of client
	if slices.Contains(p.AllowedClients, AllowAllClient) {
		// Still need to specify the client type
		if strings.TrimSpace(clientType) == "" {
			return false
		}
		return true
	}

	return slices.Contains(p.AllowedClients, clientType)
}

// validateClients checks that the given clients are not blank and there are no duplicates.
func validateClients(clients []string) error {
	foundClients := make(map[string]bool, len(clients))
	for i, clientType := range clients {
		if strings.TrimSpace(clientType) == "" {
			return fmt.Errorf("client type %d cannot be blank", i)
		}
		if foundClients[clientType] {
			return fmt.Errorf("duplicate client type: %s", clientType)
		}
		foundClients[clientType] = true
	}

	return nil
}
