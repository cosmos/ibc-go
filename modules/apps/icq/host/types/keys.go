package types

const (
	// SubModuleName defines the interchain query host module name
	SubModuleName = "icqhost"

	// StoreKey is the store key string for the interchain query host module
	StoreKey = SubModuleName
)

// ContainsQueryPath returns true if the path is present in allowQueries, otherwise false
func ContainsQueryPath(allowQueries []string, path string) bool {
	for _, v := range allowQueries {
		if v == path {
			return true
		}
	}

	return false
}
