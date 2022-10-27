package testsuite

import (
	"golang.org/x/mod/semver"
)

// FeatureReleases contains the combination of versions the feature was released in.
type FeatureReleases struct {
	// MajorVersion is the major version in the format including the v. E.g. "v6"
	MajorVersion string
	// MinorVersions contains a slice of versions including the v and excluding the patch version. E.g. v2.5
	MinorVersions []string
}

// IsSupported returns whether the version contains the feature.
// This is true if the version is greater than or equal to the major version it was released in
// or is greater than or equal to the list of minor releases it was included in.
func (fr FeatureReleases) IsSupported(versionStr string) bool {
	// assume any non-semantic version formatted version supports the feature
	// this will be useful during development of the e2e test with the new feature
	if !semver.IsValid(versionStr) {
		return true
	}

	if semverGTE(versionStr, fr.MajorVersion) {
		return true
	}

	for _, mv := range fr.MinorVersions {
		mvMajor, versionStrMajor := semver.Major(mv), semver.Major(versionStr)

		if semverEqual(mvMajor, versionStrMajor) {
			return semverGTE(versionStr, mv)
		}
	}

	return false
}

// semverGTE returns true if versionA is greater than or equal to versionB.
func semverGTE(versionA, versionB string) bool {
	return semver.Compare(versionA, versionB) >= 0
}

// semverEqual returns true if versionA is equal to versionB.
func semverEqual(versionA, versionB string) bool {
	return semver.Compare(versionA, versionB) == 0
}
