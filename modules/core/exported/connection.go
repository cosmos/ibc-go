package exported

// LocalhostConnectionID is the sentinel connection ID for the localhost connection.
const LocalhostConnectionID string = "connection-localhost"

// CounterpartyConnectionI describes the required methods for a counterparty connection.
type CounterpartyConnectionI interface {
	GetClientID() string
	GetConnectionID() string
	GetPrefix() Prefix
	ValidateBasic() error
}
