package backends_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyleterry/jot/pkg/testutil"
	"github.com/stretchr/testify/require"
)

type NoopCloseBuffer struct {
	*bytes.Buffer
}

func (cb *NoopCloseBuffer) Close() error {
	return nil
}

func TestFilesystemPut(t *testing.T) {
	tmpdir, fs, cleanup := testutil.NewTempFilesystem(t)
	defer cleanup()

	key := "abc123"
	payload := "test payload"
	buf := &NoopCloseBuffer{bytes.NewBufferString(payload)}

	require.NoError(t, fs.Put(key, buf))

	_, err := os.Stat(filepath.Join(tmpdir, key))
	require.NoError(t, err)
}

func TestFilesystemStat(t *testing.T) {
	_, fs, cleanup := testutil.NewTempFilesystem(t)
	defer cleanup()

	key := "abc123"
	payload := "test payload"
	buf := &NoopCloseBuffer{bytes.NewBufferString(payload)}

	require.NoError(t, fs.Put(key, buf))

	r, err := fs.Stat(key)
	require.NoError(t, err)

	require.NotNil(t, r.ModifiedDate)
}

func TestFilesystemGet(t *testing.T) {
	_, fs, cleanup := testutil.NewTempFilesystem(t)
	defer cleanup()

	key := "abc123"
	payload := "test payload"
	buf := &NoopCloseBuffer{bytes.NewBufferString(payload)}

	require.NoError(t, fs.Put(key, buf))

	r, err := fs.Get(key)
	require.NoError(t, err)

	responsebuf := &bytes.Buffer{}
	responsebuf.ReadFrom(r.Content)

	require.Equal(t, payload, string(responsebuf.Bytes()))
}

func TestFilesystemDelete(t *testing.T) {
	tmpdir, fs, cleanup := testutil.NewTempFilesystem(t)
	defer cleanup()

	key := "abc123"
	payload := "test payload"
	buf := &NoopCloseBuffer{bytes.NewBufferString(payload)}

	require.NoError(t, fs.Put(key, buf))
	require.NoError(t, fs.Delete(key))

	_, err := os.Stat(filepath.Join(tmpdir, key))
	require.True(t, os.IsNotExist(err))
}
