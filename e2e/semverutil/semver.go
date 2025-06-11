package semverutil

import (
	"strings"

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
	// in our compatibility tests, our images are in the format of "release-v1.0.x". We want to actually look at
	// the "1.0.x" part but we also need this to be a valid version. We can change it to "1.0.0"
	// TODO: change the way we provide the ibc-go version. This should be done in a more flexible way such
	// as docker labels/metadata instead of the tag, as this will only work for our versioning scheme.
	const releasePrefix = "release-"
	if strings.HasPrefix(versionStr, releasePrefix) {
		versionStr = versionStr[len(releasePrefix):]
		// replace x with 999 so the release version is always larger than the others in the release line.
		versionStr = strings.ReplaceAll(versionStr, "x", "999")
	}

	// assume any non-semantic version formatted version supports the feature
	// this will be useful during development of the e2e test with the new feature
	if !semver.IsValid(versionStr) {
		return true
	}

	if fr.MajorVersion != "" && GTE(versionStr, fr.MajorVersion) {
		return true
	}

	for _, mv := range fr.MinorVersions {
		mvMajor, versionStrMajor := semver.Major(mv), semver.Major(versionStr)

		if semverEqual(mvMajor, versionStrMajor) {
			return GTE(versionStr, mv)
		}
	}

	return false
}

// GTE returns true if versionA is greater than or equal to versionB.
func GTE(versionA, versionB string) bool {
	return semver.Compare(versionA, versionB) >= 0
}

// semverEqual returns true if versionA is equal to versionB.
func semverEqual(versionA, versionB string) bool {
	return semver.Compare(versionA, versionB) == 0
}
