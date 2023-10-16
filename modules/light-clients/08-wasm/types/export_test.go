package types

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

// instantiateMessage is the message that is sent to the contract's instantiate entry point.
type InstantiateMessage struct {
	instantiateMessage
}

// these fields are exported aliases for the payload fields passed to the wasm vm.
// these are used to specify callback functions to handle specific queries in the mock vm.

type StatusMsg statusMsg
type ExportMetadataMsg exportMetadataMsg
type TimestampAtHeightMsg timestampAtHeightMsg
type VerifyClientMessageMsg verifyClientMessageMsg
type CheckForMisbehaviourMsg checkForMisbehaviourMsg
