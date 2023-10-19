package types

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

// MaxWasmSize is the maximum size of a wasm code in bytes.
const MaxWasmSize = maxWasmSize

// instantiateMessage is the message that is sent to the contract's instantiate entry point.
type InstantiateMessage struct {
	instantiateMessage
}

// these fields are exported aliases for the payload fields passed to the wasm vm.
// these are used to specify callback functions to handle specific queries in the mock vm.
type (
	// Query payload types
	StatusMsg               = statusMsg
	ExportMetadataMsg       = exportMetadataMsg
	TimestampAtHeightMsg    = timestampAtHeightMsg
	VerifyClientMessageMsg  = verifyClientMessageMsg
	CheckForMisbehaviourMsg = checkForMisbehaviourMsg

	// Sudo payload types
	UpdateStateMsg                   = updateStateMsg
	UpdateStateOnMisbehaviourMsg     = updateStateOnMisbehaviourMsg
	VerifyUpgradeAndUpdateStateMsg   = verifyUpgradeAndUpdateStateMsg
	CheckSubstituteAndUpdateStateMsg = checkSubstituteAndUpdateStateMsg
	VerifyMembershipMsg              = verifyMembershipMsg
	VerifyNonMembershipMsg           = verifyNonMembershipMsg

	// Contract response types
	EmptyResult                = emptyResult
	StatusResult               = statusResult
	ExportMetadataResult       = exportMetadataResult
	TimestampAtHeightResult    = timestampAtHeightResult
	CheckForMisbehaviourResult = checkForMisbehaviourResult
	UpdateStateResult          = updateStateResult
)
