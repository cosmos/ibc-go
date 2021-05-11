package keeper

import cosmwasm "github.com/CosmWasm/wasmvm"

// Basic validation config can be extended to add other configuration later
type WASMValidationConfig struct {
	MaxSizeAllowed int
}

func NewWASMValidator(config *WASMValidationConfig, vmCreateFn func () (*cosmwasm.VM, error)) (*WASMValidator, error) {
	return &WASMValidator{
		config: config,
		vmCreateFn: vmCreateFn,
	}, nil
}

type WASMValidator struct {
	vmCreateFn func () (*cosmwasm.VM, error)
	config *WASMValidationConfig
}

func (v *WASMValidator) validateWASMCode(code []byte) (bool, error) {
	if len(code) > v.config.MaxSizeAllowed {
		return false, nil
	}

	testVm, err := v.vmCreateFn()
	if err != nil {
		return false, err
	}

	_, err = testVm.Create(code)
	if err != nil {
		return false, nil
	}

	// Validation start

	// Validation ends

	testVm.Cleanup()
	return true, nil
}


