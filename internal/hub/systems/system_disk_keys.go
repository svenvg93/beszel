package systems

import (
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// migrateExtraFsKeys moves stored extra filesystem history from legacy keys
// (device or custom names) to the UUID keys the agent now reports. Agents send
// the renames once per connection; rows that were already moved no longer
// match, so repeated runs are no-ops.
func migrateExtraFsKeys(app core.App, systemId string, renames map[string]string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		for oldKey, newKey := range renames {
			if oldKey == newKey || !isSafeJsonPathKey(oldKey) || !isSafeJsonPathKey(newKey) {
				continue
			}
			result, err := txApp.DB().NewQuery(`UPDATE system_stats
				SET stats = json_set(json_remove(stats, {:old}), {:new}, json(json_extract(stats, {:old})))
				WHERE system = {:system}
					AND json_type(stats, {:old}) IS NOT NULL
					AND json_type(stats, {:new}) IS NULL`).
				Bind(dbx.Params{
					"system": systemId,
					"old":    extraFsJsonPath(oldKey),
					"new":    extraFsJsonPath(newKey),
				}).Execute()
			if err != nil {
				return err
			}
			if rows, _ := result.RowsAffected(); rows > 0 {
				app.Logger().Info("Migrated disk history to UUID key", "system", systemId, "from", oldKey, "to", newKey, "records", rows)
			}
		}
		return nil
	})
}

// extraFsJsonPath returns the SQLite JSON path of an extra filesystem entry.
func extraFsJsonPath(key string) string {
	return `$.efs."` + key + `"`
}

// isSafeJsonPathKey reports whether a key can be quoted in a SQLite JSON path,
// which has no escape sequences for quotes.
func isSafeJsonPathKey(key string) bool {
	return key != "" && !strings.ContainsAny(key, `"\`)
}
