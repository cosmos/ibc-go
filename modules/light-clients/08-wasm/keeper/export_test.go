package keeper

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

func GenerateWasmCodeHash(code []byte) []byte {
	return generateWasmCodeHash(code)
}
