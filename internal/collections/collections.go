package collections

// Contains is a generic function which returns true if elem T exists within the list of elements []T.
func Contains[T comparable](elem T, elements []T) bool {
	for _, e := range elements {
		if elem == e {
			return true
		}
	}
	return false
}
