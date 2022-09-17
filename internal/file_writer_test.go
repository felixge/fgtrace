package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileWriter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "FileWriter")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("happy_path", func(t *testing.T) {
		name := filepath.Join(tmpDir, "happy_path.txt")
		fileDst := NewFileWriter(name)
		writes := [][]byte{[]byte("Hello "), []byte("World\n")}
		for _, w := range writes {
			n, err := fileDst.Write(w)
			require.NoError(t, err)
			require.Equal(t, len(w), n)
		}
		require.NoError(t, fileDst.Close())
		require.Error(t, fileDst.Close())
		data, err := os.ReadFile(name)
		require.NoError(t, err)
		require.Equal(t, []byte("Hello World\n"), data)
	})

	t.Run("open_error", func(t *testing.T) {
		name := filepath.Join(tmpDir, "does/not/exist.txt")
		fileDst := NewFileWriter(name)
		writes := [][]byte{[]byte("Hello "), []byte("World\n")}
		for _, w := range writes {
			n, err := fileDst.Write(w)
			require.True(t, os.IsNotExist(err))
			require.Equal(t, 0, n)
		}
		require.True(t, os.IsNotExist(fileDst.Close()))
		require.True(t, os.IsNotExist(fileDst.Close()))
		_, err := os.ReadFile(name)
		require.True(t, os.IsNotExist(err))
	})
}
