package hub

import (
	"strconv"

	"github.com/henrygd/beszel/internal/entities/speedtest"
	"github.com/henrygd/beszel/internal/hub/systems"
	"github.com/pocketbase/pocketbase/core"
)

// generateSpeedtestID creates a stable hash ID for a speedtest based on its system and server.
func generateSpeedtestID(systemID string, serverID uint32) string {
	return systems.MakeStableHashId(systemID, "speedtest", strconv.FormatUint(uint64(serverID), 10))
}

// bindSpeedtestsEvents keeps speedtest records and agent speedtest state in sync.
func bindSpeedtestsEvents(hub *Hub) {
	// on create, make sure the id is set to a stable hash
	hub.OnRecordCreate("speedtests").BindFunc(func(e *core.RecordEvent) error {
		config := speedtestConfigFromRecord(e.Record)
		e.Record.Set("id", generateSpeedtestID(e.Record.GetString("system"), config.ServerID))
		return e.Next()
	})

	// sync speedtest to agent on creation and start a first run
	hub.OnRecordAfterCreateSuccess("speedtests").BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if !e.Record.GetBool("enabled") {
			return nil
		}
		// Paused systems may be absent from the manager; their speedtests sync when they reconnect.
		system, err := hub.sm.GetSystem(e.Record.GetString("system"))
		if err == nil && system.Status == "up" {
			go func() {
				if err := hub.upsertSpeedtest(e.Record, true); err != nil {
					hub.Logger().Warn("failed to sync new speedtest", "system", system.Id, "speedtest", e.Record.Id, "err", err)
				}
			}()
		}
		return nil
	})

	// On API update requests, if the server changed, replace the record so its ID
	// stays a stable hash. Otherwise, update the speedtest on the agent.
	hub.OnRecordUpdateRequest("speedtests").BindFunc(func(e *core.RecordRequestEvent) error {
		systemID := e.Record.GetString("system")
		ID := generateSpeedtestID(systemID, speedtestConfigFromRecord(e.Record).ServerID)
		if ID != e.Record.Id {
			newRecord := core.NewRecord(e.Record.Collection())
			newRecord.Id = ID
			for _, field := range []string{"system", "server_id", "server_name", "server_location", "interval", "enabled"} {
				newRecord.Set(field, e.Record.Get(field))
			}
			if err := e.App.Save(newRecord); err != nil {
				return err
			}
			return e.App.Delete(e.Record)
		}
		if err := e.Next(); err != nil {
			return err
		}
		var err error
		if e.Record.GetBool("enabled") {
			runNow := !e.Record.Original().GetBool("enabled")
			err = hub.upsertSpeedtest(e.Record, runNow)
		} else {
			err = hub.deleteSpeedtest(e.Record)
		}
		if err != nil {
			hub.Logger().Warn("failed to sync updated speedtest", "system", systemID, "speedtest", e.Record.Id, "err", err)
		}
		return nil
	})

	// sync speedtest to agent on delete
	hub.OnRecordAfterDeleteSuccess("speedtests").BindFunc(func(e *core.RecordEvent) error {
		if err := hub.deleteSpeedtest(e.Record); err != nil {
			hub.Logger().Warn("failed to delete speedtest on agent", "system", e.Record.GetString("system"), "speedtest", e.Record.Id, "err", err)
		}
		return e.Next()
	})
}

// speedtestConfigFromRecord builds a speedtest config from a speedtests record.
func speedtestConfigFromRecord(record *core.Record) speedtest.Config {
	return speedtest.Config{
		ID:       record.Id,
		ServerID: uint32(max(record.GetInt("server_id"), 0)),
		Interval: uint32(max(record.GetInt("interval"), speedtest.MinInterval)),
	}
}

// upsertSpeedtest creates or updates the record's speedtest on its system's agent.
func (h *Hub) upsertSpeedtest(record *core.Record, runNow bool) error {
	system, err := h.sm.GetSystem(record.GetString("system"))
	if err != nil {
		return err
	}
	return system.UpsertSpeedtest(speedtestConfigFromRecord(record), runNow)
}

// deleteSpeedtest removes the record's speedtest from its system's agent.
func (h *Hub) deleteSpeedtest(record *core.Record) error {
	system, err := h.sm.GetSystem(record.GetString("system"))
	if err != nil {
		return err
	}
	return system.DeleteSpeedtest(record.Id)
}
