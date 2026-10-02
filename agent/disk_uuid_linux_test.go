//go:build linux && testing

package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFsUUIDs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Symlink("../../sdb1", filepath.Join(dir, "0a1b2c3d-aaaa-bbbb-cccc-111122223333")))
	require.NoError(t, os.Symlink("../../dm-3", filepath.Join(dir, "4E21-1A2B")))
	old := fsUUIDDir
	fsUUIDDir = dir
	t.Cleanup(func() { fsUUIDDir = old })

	assert.Equal(t, map[string]string{
		"sdb1": "0a1b2c3d-aaaa-bbbb-cccc-111122223333",
		"dm-3": "4E21-1A2B",
	}, readFsUUIDs())

	fsUUIDDir = filepath.Join(dir, "missing")
	assert.Nil(t, readFsUUIDs())
}
