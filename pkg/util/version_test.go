package util_test

import (
	"testing"

	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestVersionString(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	_, err := util.VersionString()
	a.Nil(err)
}

func TestVersionStringBadRelease(t *testing.T) {
	oldVal := util.Release
	defer func() {
		util.Release = oldVal
	}()
	util.Release = "some random value"
	a := assert.New(t)
	_, err := util.VersionString()
	a.NotNil(err)
}

func TestVersionStringBadDirty(t *testing.T) {
	oldVal := util.Dirty
	defer func() {
		util.Dirty = oldVal
	}()
	util.Dirty = "some random value"
	a := assert.New(t)
	_, err := util.VersionString()
	a.NotNil(err)
}

func TestVersionCacheKeyError(t *testing.T) {
	oldVal := util.Dirty
	defer func() {
		util.Dirty = oldVal
	}()
	util.Dirty = "some random value"

	a := assert.New(t)
	s := util.VersionCacheKey()
	a.Empty(s)
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
	a := assert.New(t)
	s := util.VersionCacheKey()
	a.Equal(util.Version, s)
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
	a := assert.New(t)
	s := util.VersionCacheKey()
	a.Empty(s)
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
	a := assert.New(t)
	s := util.VersionCacheKey()
	a.Equal("1.1.1", s)
}
