package util_test

import (
	"testing"

	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestVersionString(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	_, err := util.VersionString()
	r.Nil(err)
}

func TestVersionStringBadRelease(t *testing.T) {
	oldVal := util.Release
	defer func() {
		util.Release = oldVal
	}()
	util.Release = "some random value"
	r := require.New(t)
	_, err := util.VersionString()
	r.NotNil(err)
}

func TestVersionStringBadDirty(t *testing.T) {
	oldVal := util.Dirty
	defer func() {
		util.Dirty = oldVal
	}()
	util.Dirty = "some random value"
	r := require.New(t)
	_, err := util.VersionString()
	r.NotNil(err)
}

func TestVersionCacheKeyError(t *testing.T) {
	oldVal := util.Dirty
	defer func() {
		util.Dirty = oldVal
	}()
	util.Dirty = "some random value"

	r := require.New(t)
	s := util.VersionCacheKey()
	r.Empty(s)
}

func TestVersionCacheKey(t *testing.T) {
	oldVersion := util.Version
	oldSha := util.GitSha
	oldRelease := util.Release
	oldDirty := util.Dirty
	defer func() {
		util.Version = oldVersion
		util.GitSha = oldSha
		util.Release = oldRelease
		util.Dirty = oldDirty
	}()
	util.Version = "1.1.1"
	util.GitSha = "gitsha"
	util.Release = "true"
	util.Dirty = "false"
	r := require.New(t)
	s := util.VersionCacheKey()
	r.Equal(util.Version, s)
}

func TestVersionCacheKeyBadVersion(t *testing.T) {
	oldVersion := util.Version
	oldSha := util.GitSha
	oldRelease := util.Release
	oldDirty := util.Dirty
	defer func() {
		util.Version = oldVersion
		util.GitSha = oldSha
		util.Release = oldRelease
		util.Dirty = oldDirty
	}()
	util.Version = "bad-1.1.1"
	util.GitSha = "gitsha"
	util.Release = "true"
	util.Dirty = "false"
	r := require.New(t)
	s := util.VersionCacheKey()
	r.Empty(s)
}

func TestVersionCacheKeyDirty(t *testing.T) {
	oldVersion := util.Version
	oldSha := util.GitSha
	oldRelease := util.Release
	oldDirty := util.Dirty
	defer func() {
		util.Version = oldVersion
		util.GitSha = oldSha
		util.Release = oldRelease
		util.Dirty = oldDirty
	}()
	util.Version = "1.1.1"
	util.GitSha = "gitsha"
	util.Release = "false"
	util.Dirty = "false"
	r := require.New(t)
	s := util.VersionCacheKey()
	r.Equal("1.1.1", s)
}
