package types

const (
	// DefaultControllerEnabled is the default value for the controller param (set to true)
	DefaultControllerEnabled = true
)

// KeyControllerEnabled is the store key for ControllerEnabled Params
var KeyControllerEnabled = []byte("ControllerEnabled")

// NewParams creates a new parameter configuration for the controller submodule
func NewParams(enableController bool) Params {
	return Params{
		ControllerEnabled: enableController,
	}
}

// DefaultParams is the default parameter configuration for the controller submodule
func DefaultParams() Params {
	return NewParams(DefaultControllerEnabled)
}

