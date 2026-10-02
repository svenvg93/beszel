//go:build testing

package agent

import (
	"os"
	"testing"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSymlinks resolves the given device paths and returns an error otherwise.
func fakeSymlinks(links map[string]string) func(string) (string, error) {
	return func(path string) (string, error) {
		if target, ok := links[path]; ok {
			return target, nil
		}
		return "", os.ErrNotExist
	}
}

func TestUuidFor(t *testing.T) {
	setEvalSymlinks(t, fakeSymlinks(map[string]string{"/dev/mapper/vg-data": "/dev/dm-3"}))
	d := diskDiscovery{uuids: map[string]string{"sdb1": "uuid-sdb1", "dm-3": "uuid-lvm", "md0": "uuid-md"}}

	assert.Equal(t, "uuid-sdb1", d.uuidFor("/dev/sdb1", "sdb1"))
	assert.Equal(t, "uuid-sdb1", d.uuidFor("sdb1", ""), "bare folder device name")
	assert.Equal(t, "uuid-lvm", d.uuidFor("/dev/mapper/vg-data", "vg-data"), "symlink target")
	assert.Equal(t, "uuid-md", d.uuidFor("/dev/md/storage", "md0"), "I/O key fallback")
	assert.Empty(t, d.uuidFor("/dev/sdc1", "sdc1"))
	assert.Empty(t, d.uuidFor("/dev/sdc1", ""), "empty I/O key never matches")
	assert.Empty(t, (&diskDiscovery{}).uuidFor("/dev/sdb1", "sdb1"), "no UUID map")
}

func TestIsAutoDiscoverCandidate(t *testing.T) {
	tests := []struct {
		name string
		p    disk.PartitionStat
		want bool
	}{
		{"data disk", disk.PartitionStat{Device: "/dev/sdb1", Mountpoint: "/mnt/data", Fstype: "ext4"}, true},
		{"lvm volume", disk.PartitionStat{Device: "/dev/mapper/vg-data", Mountpoint: "/srv", Fstype: "xfs"}, true},
		{"root", disk.PartitionStat{Device: "/dev/sda2", Mountpoint: "/", Fstype: "ext4"}, false},
		{"tmpfs", disk.PartitionStat{Device: "tmpfs", Mountpoint: "/tmp", Fstype: "tmpfs"}, false},
		{"nfs", disk.PartitionStat{Device: "nas:/export", Mountpoint: "/mnt/nas", Fstype: "nfs4"}, false},
		{"zfs dataset", disk.PartitionStat{Device: "tank/data", Mountpoint: "/tank/data", Fstype: "zfs"}, false},
		{"snap loop", disk.PartitionStat{Device: "/dev/loop3", Mountpoint: "/snap/core/1", Fstype: "squashfs"}, false},
		{"squashfs", disk.PartitionStat{Device: "/dev/sdc1", Mountpoint: "/media/iso", Fstype: "squashfs"}, false},
		{"cdrom", disk.PartitionStat{Device: "/dev/sr0", Mountpoint: "/media/cdrom", Fstype: "udf"}, false},
		{"efi", disk.PartitionStat{Device: "/dev/sda1", Mountpoint: "/boot/efi", Fstype: "vfat"}, false},
		{"boot", disk.PartitionStat{Device: "/dev/sda1", Mountpoint: "/boot", Fstype: "ext4"}, false},
		{"docker data", disk.PartitionStat{Device: "/dev/sdb1", Mountpoint: "/var/lib/docker", Fstype: "ext4"}, false},
		{"docker hosts file", disk.PartitionStat{Device: "/dev/sda2", Mountpoint: "/etc/hosts", Fstype: "ext4"}, false},
		{"extra-filesystems mount", disk.PartitionStat{Device: "/dev/sdb1", Mountpoint: "/extra-filesystems/sdb1", Fstype: "ext4"}, false},
		{"similar prefix is not skipped", disk.PartitionStat{Device: "/dev/sdb1", Mountpoint: "/bootstrap", Fstype: "ext4"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isAutoDiscoverCandidate(tt.p, "/", "/extra-filesystems"))
		})
	}
}

func TestIsExcludedFilesystem(t *testing.T) {
	patterns := []string{"/mnt/backup", "sdc*", "1234-ABCD"}
	assert.True(t, isExcludedFilesystem(patterns, "/mnt/backup", "/dev/sdb1", "x"))
	assert.True(t, isExcludedFilesystem(patterns, "/mnt/other", "/dev/sdc2", "x"))
	assert.True(t, isExcludedFilesystem(patterns, "/mnt/other", "/dev/sdd1", "1234-ABCD"))
	assert.False(t, isExcludedFilesystem(patterns, "/mnt/data", "/dev/sdb1", "x"))
	assert.False(t, isExcludedFilesystem(nil, "/mnt/data", "/dev/sdb1", "x"))
}

func newAutoDiscovery(partitions []disk.PartitionStat, uuids map[string]string, ioDevices ...string) (*Agent, *diskDiscovery) {
	agent := &Agent{fsStats: make(map[string]*system.FsStats)}
	counters := make(map[string]disk.IOCountersStat, len(ioDevices))
	for _, name := range ioDevices {
		counters[name] = disk.IOCountersStat{Name: name}
	}
	return agent, &diskDiscovery{
		agent:          agent,
		rootMountPoint: "/",
		partitions:     partitions,
		uuids:          uuids,
		ctx: fsRegistrationContext{
			efPath:         "/extra-filesystems",
			diskIoCounters: counters,
		},
	}
}

func TestAddAutoDiscoveredFs(t *testing.T) {
	setEvalSymlinks(t, fakeSymlinks(nil))

	t.Run("adds data disks keyed by I/O device with UUID and mountpoint label", func(t *testing.T) {
		agent, d := newAutoDiscovery([]disk.PartitionStat{
			{Device: "/dev/sda2", Mountpoint: "/", Fstype: "ext4"},
			{Device: "/dev/sdb1", Mountpoint: "/mnt/data", Fstype: "ext4"},
			{Device: "tmpfs", Mountpoint: "/run", Fstype: "tmpfs"},
		}, map[string]string{"sda2": "uuid-root", "sdb1": "uuid-data"}, "sda", "sda2", "sdb1")

		d.addAutoDiscoveredFs()

		require.Len(t, agent.fsStats, 1)
		stats := agent.fsStats["sdb1"]
		require.NotNil(t, stats)
		assert.Equal(t, "uuid-data", stats.UUID)
		assert.Equal(t, "/mnt/data", stats.Label)
		assert.Equal(t, "/mnt/data", stats.Mountpoint)
		assert.False(t, stats.Root)
		assert.Empty(t, stats.Name)
	})

	t.Run("tracks a filesystem mounted twice once, at the shortest mountpoint", func(t *testing.T) {
		agent, d := newAutoDiscovery([]disk.PartitionStat{
			{Device: "/dev/sdb1", Mountpoint: "/mnt/pool/snapshots", Fstype: "btrfs"},
			{Device: "/dev/sdb1", Mountpoint: "/mnt/pool", Fstype: "btrfs"},
		}, map[string]string{"sdb1": "uuid-pool"}, "sdb1")

		d.addAutoDiscoveredFs()

		require.Len(t, agent.fsStats, 1)
		assert.Equal(t, "/mnt/pool", agent.fsStats["sdb1"].Mountpoint)
	})

	t.Run("skips the root filesystem's UUID", func(t *testing.T) {
		agent, d := newAutoDiscovery([]disk.PartitionStat{
			{Device: "/dev/sda2", Mountpoint: "/", Fstype: "ext4"},
			{Device: "/dev/sda2", Mountpoint: "/var/lib/beszel-agent", Fstype: "ext4"},
		}, map[string]string{"sda2": "uuid-root"}, "sda2")

		d.addAutoDiscoveredFs()

		assert.Empty(t, agent.fsStats)
	})

	t.Run("does not duplicate a configured disk", func(t *testing.T) {
		agent, d := newAutoDiscovery([]disk.PartitionStat{
			{Device: "/dev/sdb1", Mountpoint: "/mnt/backup", Fstype: "ext4"},
		}, map[string]string{"sdb1": "uuid-backup"}, "sdb1")
		d.addFsStat("/dev/sdb1", "/mnt/backup", false, "backup")

		d.addAutoDiscoveredFs()

		require.Len(t, agent.fsStats, 1)
		stats := agent.fsStats["sdb1"]
		assert.Equal(t, "backup", stats.Name)
		assert.Equal(t, "backup", stats.Label, "configured disks keep their name as label")
		assert.Equal(t, "uuid-backup", stats.UUID)
	})

	t.Run("skips disks without a UUID", func(t *testing.T) {
		agent, d := newAutoDiscovery([]disk.PartitionStat{
			{Device: "/dev/sdb1", Mountpoint: "/mnt/data", Fstype: "ext4"},
		}, map[string]string{"sdc1": "uuid-other"}, "sdb1")

		d.addAutoDiscoveredFs()

		assert.Empty(t, agent.fsStats)
	})

	t.Run("honors EXCLUDE_FILESYSTEMS", func(t *testing.T) {
		agent, d := newAutoDiscovery([]disk.PartitionStat{
			{Device: "/dev/sdb1", Mountpoint: "/mnt/data", Fstype: "ext4"},
			{Device: "/dev/sdc1", Mountpoint: "/mnt/scratch", Fstype: "ext4"},
			{Device: "/dev/sdd1", Mountpoint: "/mnt/backup", Fstype: "ext4"},
			{Device: "/dev/sde1", Mountpoint: "/mnt/media", Fstype: "ext4"},
		}, map[string]string{"sdb1": "uuid-b", "sdc1": "uuid-c", "sdd1": "uuid-d", "sde1": "uuid-e"},
			"sdb1", "sdc1", "sdd1", "sde1")
		d.excludes = []string{"/mnt/backup", "sdc*", "uuid-e"}

		d.addAutoDiscoveredFs()

		require.Len(t, agent.fsStats, 1)
		assert.Contains(t, agent.fsStats, "sdb1")
	})
}

func TestExtraFsOutputKeys(t *testing.T) {
	agent := &Agent{
		fsStats: map[string]*system.FsStats{
			"sda2":   {Root: true, UUID: "uuid-root", Mountpoint: "/"},
			"sdb1":   {UUID: "uuid-b", Label: "backup", Name: "backup", DiskTotal: 100, DiskUsed: 25},
			"sdc1":   {UUID: "uuid-c", Label: "/mnt/data", DiskTotal: 200, DiskUsed: 50},
			"nouuid": {Label: "nouuid", DiskTotal: 10, DiskUsed: 1},
		},
	}

	assert.Equal(t, "uuid-b", extraFsKey("sdb1", agent.fsStats["sdb1"], true))
	assert.Equal(t, "backup", extraFsKey("sdb1", agent.fsStats["sdb1"], false))
	assert.Equal(t, "sdc1", extraFsKey("sdc1", agent.fsStats["sdc1"], false))
	assert.Equal(t, "nouuid", extraFsKey("nouuid", agent.fsStats["nouuid"], true))

	assert.Equal(t, map[string]string{"backup": "uuid-b", "sdc1": "uuid-c"}, agent.extraFsKeyRenames())
}

func TestAttachSystemDetailsFsKeyRenames(t *testing.T) {
	agent := &Agent{fsStats: map[string]*system.FsStats{
		"sdb1": {UUID: "uuid-b", Label: "sdb1"},
	}}

	legacy := agent.attachSystemDetails(&system.CombinedData{}, defaultDataCacheTimeMs, true, false)
	require.NotNil(t, legacy.Details)
	assert.Nil(t, legacy.Details.FsKeyRenames)

	withUUIDs := agent.attachSystemDetails(&system.CombinedData{}, defaultDataCacheTimeMs, true, true)
	require.NotNil(t, withUUIDs.Details)
	assert.Equal(t, map[string]string{"sdb1": "uuid-b"}, withUUIDs.Details.FsKeyRenames)
	assert.Nil(t, agent.systemDetails.FsKeyRenames, "the stored details are not modified")
}
