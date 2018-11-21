package util

import (
	"fmt"
	"strconv"

	"github.com/blang/semver"
	"github.com/pkg/errors"
)

var (
	// Version is the blessclient version
	Version = "undefined"
	// GitSha is the gitsha used to build this version
	GitSha = "undefined"
	// Release is true if this is a release
	Release = "false"
	// Dirty if git is dirty
	Dirty = "true"
)

// VersionString returns the version string
func VersionString() (string, error) {
	release, e := strconv.ParseBool(Release)
	if e != nil {
		return "", errors.Wrapf(e, "unable to parse version release field %s", Release)
	}
	dirty, e := strconv.ParseBool(Dirty)
	if e != nil {
		return "", errors.Wrapf(e, "unable to parse version dirty field %s", Dirty)
	}
	return versionString(Version, GitSha, release, dirty), nil
}

// VersionCacheKey returns a key to version the cache
func VersionCacheKey() string {
	versionString, e := VersionString()
	if e != nil {
		return ""
	}
	v, e := semver.Parse(versionString)
	if e != nil {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func versionString(version, sha string, release, dirty bool) string {
	if release {
		return version
	}
	if !dirty {
		return fmt.Sprintf("%s+%s", version, sha)
	}
	return fmt.Sprintf("%s+%s-dirty", version, sha)
}
