package types

import (
	"fmt"
	"strings"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

<<<<<<< HEAD
var (
	// DefaultAllowedClients are the default clients for the AllowedClients parameter.
	DefaultAllowedClients = []string{exported.Solomachine, exported.Tendermint, exported.Localhost}

	// KeyAllowedClients is store's key for AllowedClients Params
	KeyAllowedClients = []byte("AllowedClients")
)

// ParamKeyTable type declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}
=======
// Maximum length of the allowed clients list
const MaxAllowedClientsLength = 200

// DefaultAllowedClients are the default clients for the AllowedClients parameter.
// By default it allows all client types.
var DefaultAllowedClients = []string{AllowAllClients}
>>>>>>> 478f4c60 (imp: check length of slices of messages (#6256))

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

// Validate all ibc-client module parameters
func (p Params) Validate() error {
	return validateClients(p.AllowedClients)
}

// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyAllowedClients, p.AllowedClients, validateClients),
	}
}

<<<<<<< HEAD
// IsAllowedClient checks if the given client type is registered on the allowlist.
func (p Params) IsAllowedClient(clientType string) bool {
	for _, allowedClient := range p.AllowedClients {
		if allowedClient == clientType {
			return true
		}
	}
	return false
}

func validateClients(i interface{}) error {
	clients, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
=======
// validateClients checks that the given clients are not blank and there are no duplicates.
// If AllowAllClients wildcard (*) is used, then there should no other client types in the allow list
func validateClients(clients []string) error {
	if len(clients) > MaxAllowedClientsLength {
		return fmt.Errorf("allowed clients length must not exceed %d items", MaxAllowedClientsLength)
	}

	if slices.Contains(clients, AllowAllClients) && len(clients) > 1 {
		return fmt.Errorf("allow list must have only one element because the allow all clients wildcard (%s) is present", AllowAllClients)
>>>>>>> 478f4c60 (imp: check length of slices of messages (#6256))
	}

	for i, clientType := range clients {
		if strings.TrimSpace(clientType) == "" {
			return fmt.Errorf("client type %d cannot be blank", i)
		}
	}

	return nil
}
