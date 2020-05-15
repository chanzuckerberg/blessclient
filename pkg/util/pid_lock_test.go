package util_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/cenkalti/backoff"

	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LockTestSuite struct {
	suite.Suite

	lockDir string
}

func (ts *LockTestSuite) SetupTest() {
	t := ts.T()
	r := require.New(t)

	dirname, err := ioutil.TempDir("", "blessclient-lock-test")
	r.Nil(err)
	ts.lockDir = dirname
}
func (ts *LockTestSuite) TearDownTest() {
	os.RemoveAll(ts.lockDir)
}

func (ts *LockTestSuite) TestNewLock() {
	t := ts.T()
	r := require.New(t)
	l, err := util.NewLock(ts.lockDir)
	r.Nil(err)
	r.NotNil(l)
}

func (ts *LockTestSuite) TestNewLockRelativePath() {
	t := ts.T()
	r := require.New(t)
	l, err := util.NewLock("foo/bar")
	r.NotNil(err)
	r.Contains(err.Error(), "foo/bar must be an absolute path")
	r.Nil(l)
}

// We spawn another process while we hold the lock to make sure it cannot acquire it
func TestLock(t *testing.T) {
	r := require.New(t)
	lockDir := os.Getenv("TEST_LOCK_DIR")
	var err error
	// only create the lock if we're not the failing process
	if os.Getenv("SHOULD_FAIL_LOCK") != "YES" {
		lockDir, err = ioutil.TempDir("", "blessclient-lock-test")
		r.Nil(err)
		defer os.RemoveAll(lockDir)
	}

	l, err := util.NewLock(lockDir)
	r.Nil(err)
	// nolint: gosimple
	var b backoff.BackOff
	b = backoff.NewConstantBackOff(time.Millisecond)
	// Only one retry so we don't wait too long
	b = backoff.WithMaxRetries(b, uint64(1))

	err = l.Lock(b)
	// nolint: errcheck
	defer l.Unlock()
	if os.Getenv("SHOULD_FAIL_LOCK") == "YES" {
		// This lock should fail since the other process owns it
		r.NotNil(err)
		return
	}
	r.Nil(err)

	// while still locked spawn another process
	cmd := exec.Command(os.Args[0], "-test.run=TestLock")
	cmd.Env = append(
		os.Environ(),
		"SHOULD_FAIL_LOCK=YES",
		fmt.Sprintf("TEST_LOCK_DIR=%s", lockDir))
	err = cmd.Run()
	// Expect no error since we already did an assertion on error in the other process
	r.Nil(err)
}

func TestLockTestSuite(t *testing.T) {
	suite.Run(t, &LockTestSuite{})
}
