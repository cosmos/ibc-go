package types

// NewNegotiateAppVersionResponse creates a new NegotiateAppVersionResponse instance
func NewNegotiateAppVersionResponse(portID, version string) *NegotiateAppVersionResponse {
	return &NegotiateAppVersionResponse{
		PortId:  portID,
		Version: version,
	}
}
