package types

// ValidateIBCDenom is a wrapper around validateIBCDenom for testing purposes.
func ValidateIBCDenom(denom string) error {
	return validateIBCDenom(denom)
}
