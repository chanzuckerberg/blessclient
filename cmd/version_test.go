package cmd

import (
	"testing"

	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestVersionNoError(t *testing.T) {
	a := assert.New(t)
	err := versionCmd.RunE(nil, nil)
	a.Nil(err)
}

func TestVersionError(t *testing.T) {
	a := assert.New(t)
	oldRelease := util.Release
	defer func() {
		util.Release = oldRelease
	}()
	util.Release = "An Invalid Release"

	err := versionCmd.RunE(nil, nil)
	a.NotNil(err)
}
