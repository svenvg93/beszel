//go:build linux

package agent

import (
	"os"
	"path/filepath"
)

// fsUUIDDir holds udev's filesystem UUID symlinks; it is a variable so tests
// can point it at a fixture directory.
var fsUUIDDir = "/dev/disk/by-uuid"

// readFsUUIDs maps block device names (e.g. sdb1, dm-3) to filesystem UUIDs.
// Only the symlink targets are read, so the device nodes themselves need not
// exist (e.g. when the directory is bind-mounted into a container).
func readFsUUIDs() map[string]string {
	entries, err := os.ReadDir(fsUUIDDir)
	if err != nil {
		return nil
	}
	uuids := make(map[string]string, len(entries))
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join(fsUUIDDir, entry.Name()))
		if err != nil {
			continue
		}
		uuids[filepath.Base(target)] = entry.Name()
	}
	return uuids
}
