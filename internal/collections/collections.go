package collections

func Contains[T comparable](elem T, elements []T) bool {
	for _, e := range elements {
		if elem == e {
			return true
		}
	}
	return false
}
