package testsuite

import (
	"strconv"
	"strings"
)

// Version follows semantic versioning while disregarding the patch version.
type Version struct {
	Major uint64
	Minor uint64
}

// NewVersion constructs a new version given the major and minor version.
func NewVersion(major, minor uint64) Version {
	return Version{
		Major: major,
		Minor: minor,
	}
}

func ParseChainVersion(version string) (Version, bool) {
	// remove any "v" label
	version = strings.TrimPrefix(version, "v")

	versions := strings.Split(version, ".")
	if len(versions) != 3 {
		return Version{}, false
	}

	major, err := strconv.ParseUint(versions[0], 10, 64)
	if err != nil {
		return Version{}, false
	}

	minor, err := strconv.ParseUint(versions[1], 10, 64)
	if err != nil {
		return Version{}, false
	}

	return Version{
		Major: major,
		Minor: minor,
	}, true
}

// FeatureReleases contains the combination of versions the feature was released in.
type FeatureReleases struct {
	MajorVersion  uint64
	MinorVersions []Version
}

// IsSupported returns whether the version contains the feature.
// This is true if the version is greater than or equal to the major version it was released in
// or is greater than or equal to the list of minor releases it was included in.
func (fr FeatureReleases) IsSupported(versionStr string) bool {
	// assume any non semantic version formatted version support the feature
	// this will be useful during development of the e2e test with the new feature
	version, ok := ParseChainVersion(versionStr)
	if !ok {
		return true
	}

	if version.Major >= fr.MajorVersion {
		return true
	}

	for _, mv := range fr.MinorVersions {
		if mv.Major == version.Major {
			return version.Minor >= mv.Minor
		}
	}

	return false
}
