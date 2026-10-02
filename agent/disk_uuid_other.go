//go:build !linux

package agent

// readFsUUIDs is only implemented on Linux.
func readFsUUIDs() map[string]string {
	return nil
}
