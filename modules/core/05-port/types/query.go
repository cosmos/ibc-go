package types

// NewQueryPortResponse creates a new QueryPortResponse instance
func NewQueryPortResponse(portID, version string) *QueryPortResponse {
	return &QueryPortResponse{
		PortId:  portID,
		Version: version,
	}
}
