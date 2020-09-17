package testutil

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/kyleterry/jot/pkg/jot/store/backends"
	"github.com/stretchr/testify/require"
)

// NewTempFilesystem returns a backends.Filesystem configured with a temporary
// directory and a cleanup callback function.
func NewTempFilesystem(t *testing.T) (string, *backends.Filesystem, func()) {
	tmp, err := ioutil.TempDir("", "github.com-kyleterry-jot")
	require.NoError(t, err)

	fs, err := backends.NewFilesystem(backends.FilesystemOptions{Path: tmp})
	require.NoError(t, err)

	return tmp, fs, func() {
		require.NoError(t, os.RemoveAll(tmp))
	}
}
