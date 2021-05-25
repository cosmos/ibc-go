package keeper

import cosmwasm "github.com/CosmWasm/wasmvm"

// Basic validation config can be extended to add other configuration later
type ValidationConfig struct {
	MaxSizeAllowed int
}

func NewWASMValidator(config *ValidationConfig, vmCreateFn func() (*cosmwasm.VM, error)) (*WASMValidator, error) {
	return &WASMValidator{
		config:     config,
		vmCreateFn: vmCreateFn,
	}, nil
}

type WASMValidator struct {
	vmCreateFn func() (*cosmwasm.VM, error)
	config     *ValidationConfig
}

func (v *WASMValidator) validateWASMCode(code []byte) (bool, error) {
	if len(code) > v.config.MaxSizeAllowed {
		return false, nil
	}

	testVM, err := v.vmCreateFn()
	if err != nil {
		return false, err
	}

	_, err = testVM.Create(code)
	if err != nil {
		return false, nil
	}

	testVM.Cleanup()
	return true, nil
}
