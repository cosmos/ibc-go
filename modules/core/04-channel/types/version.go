package types

import "strings"

// SplitChannelVersion middleware version will split the channel version string
// into the outermost middleware version and the underlying app version.
// It will use the default delimiter `:` for middleware versions.
// In case there's no delimeter, this function returns an empty string for the middleware version (first return argument),
// and the full input as the second underlying app version.
func SplitChannelVersion(version string) (middlewareVersion, appVersion string) {
	// only split out the first middleware version
	splitVersions := strings.Split(version, ":")
	if len(splitVersions) == 1 {
		return "", version
	}
	middlewareVersion = splitVersions[0]
	appVersion = strings.Join(splitVersions[1:], ":")
	return
}
