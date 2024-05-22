package types

/*
	This file is to allow for unexported functions to be accessible to the testing package.
*/

// GetCallbackAddress is a wrapper around getCallbackAddress to allow the function to be directly called in tests.
func GetCallbackAddress(callbackData map[string]interface{}) string {
	return getCallbackAddress(callbackData)
}

// GetUserDefinedGasLimit is a wrapper around getUserDefinedGasLimit to allow the function to be directly called in tests.
func GetUserDefinedGasLimit(callbackData map[string]interface{}) uint64 {
	return getUserDefinedGasLimit(callbackData)
}
