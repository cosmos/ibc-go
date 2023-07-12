package types

// MetadataFromVersion attempts to parse the given string into a fee version Metadata,
// an error is returned if it fails to do so.
func MetadataFromVersion(version string) (Metadata, error) {
	var metadata Metadata
	err := ModuleCdc.UnmarshalJSON([]byte(version), &metadata)
	if err != nil {
		return Metadata{}, ErrInvalidVersion
	}

	return metadata, nil
}
