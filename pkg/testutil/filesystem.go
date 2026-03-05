package testutil

import (
	"os"
	"testing"

	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/jot/store/backends"
	"github.com/stretchr/testify/require"
)

// NewTextFilesystem returns a backends.Filesystem configured with a temporary
// directory and a cleanup callback function.
func NewTextFilesystem(t *testing.T) (string, *backends.Filesystem, func()) {
	tmp, err := os.MkdirTemp("", "github.com-kyleterry-jot")
	require.NoError(t, err)

	fs, err := backends.NewFilesystem(backends.FilesystemOptions{
		Path:                 tmp,
		DirectoryPermissions: config.DirectoryPermissions,
		FilePermissions:      config.FilePermissions,
	})
	require.NoError(t, err)

	return tmp, fs, func() {
		require.NoError(t, os.RemoveAll(tmp))
	}
}
