//go:build testing

package systems_test

import (
	"encoding/json"
	"testing"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/hub/systems"
	"github.com/henrygd/beszel/internal/tests"
	"github.com/pocketbase/dbx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateExtraFsKeys(t *testing.T) {
	hub, user := tests.GetHubWithUser(t)
	defer hub.Cleanup()
	records, err := tests.CreateSystems(hub, 2, user.Id, "paused")
	require.NoError(t, err)
	systemId, otherId := records[0].Id, records[1].Id

	addStats := func(sysId string, efs map[string]*system.FsStats) {
		_, err := tests.CreateRecord(hub, "system_stats", map[string]any{
			"system": sysId,
			"type":   "1m",
			"stats":  system.Stats{Cpu: 1, ExtraFs: efs},
		})
		require.NoError(t, err)
	}
	addStats(systemId, map[string]*system.FsStats{"sdb1": {DiskTotal: 100, DiskUsed: 40}, "other": {DiskTotal: 5}})
	addStats(systemId, map[string]*system.FsStats{"sdb1": {DiskTotal: 100, DiskUsed: 50}})
	addStats(systemId, map[string]*system.FsStats{"uuid-1": {DiskTotal: 100, DiskUsed: 60, Label: "sdb1"}})
	addStats(otherId, map[string]*system.FsStats{"sdb1": {DiskTotal: 200, DiskUsed: 10}})

	loadEfs := func(sysId string) []map[string]system.FsStats {
		var rows []struct {
			Stats string `db:"stats"`
		}
		require.NoError(t, hub.DB().Select("stats").From("system_stats").
			Where(dbx.HashExp{"system": sysId}).OrderBy("rowid").All(&rows))
		out := make([]map[string]system.FsStats, 0, len(rows))
		for _, row := range rows {
			var stats system.Stats
			require.NoError(t, json.Unmarshal([]byte(row.Stats), &stats))
			efs := make(map[string]system.FsStats, len(stats.ExtraFs))
			for k, v := range stats.ExtraFs {
				efs[k] = *v
			}
			out = append(out, efs)
		}
		return out
	}

	renames := map[string]string{"sdb1": "uuid-1", `bad"key`: "uuid-2"}
	require.NoError(t, systems.MigrateExtraFsKeys(hub, systemId, renames))

	efs := loadEfs(systemId)
	require.Len(t, efs, 3)
	assert.Equal(t, 40.0, efs[0]["uuid-1"].DiskUsed)
	assert.Contains(t, efs[0], "other", "unrelated keys stay")
	assert.Equal(t, 50.0, efs[1]["uuid-1"].DiskUsed)
	assert.Equal(t, 60.0, efs[2]["uuid-1"].DiskUsed)
	for _, row := range efs {
		assert.NotContains(t, row, "sdb1")
	}
	assert.Contains(t, loadEfs(otherId)[0], "sdb1", "other systems are untouched")

	// Running again changes nothing.
	require.NoError(t, systems.MigrateExtraFsKeys(hub, systemId, renames))
	assert.Equal(t, efs, loadEfs(systemId))
}
