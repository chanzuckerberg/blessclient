package cmd

import (
	"testing"

	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestVersionNoError(t *testing.T) {
	r := require.New(t)
	err := versionCmd.RunE(nil, nil)
	r.Nil(err)
}

func TestVersionError(t *testing.T) {
	r := require.New(t)
	oldRelease := util.Release
	defer func() {
		util.Release = oldRelease
	}()
	util.Release = "An Invalid Release"

	err := versionCmd.RunE(nil, nil)
	r.NotNil(err)
}
