package migrations

import (
	"ganigeorgiev.com/pocketbase/pocketbase/core"
	m "ganigeorgiev.com/pocketbase/pocketbase/migrations"
)

// Jitter is stored as a sum and count, like res_sum and success_count, so it
// keeps its weight through every retention tier.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("network_monitor_stats")
		if err != nil {
			return err
		}
		collection.Fields.Add(&core.NumberField{
			Id:      "nms_jitter_sum",
			Name:    "jitter_sum",
			Help:    "Sum of RTT differences between consecutive replies in microseconds",
			Min:     new(float64),
			OnlyInt: true,
		})
		collection.Fields.Add(&core.NumberField{
			Id:      "nms_jitter_count",
			Name:    "jitter_count",
			Help:    "Number of RTT differences in jitter_sum",
			Min:     new(float64),
			OnlyInt: true,
		})
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("network_monitor_stats")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("jitter_sum")
		collection.Fields.RemoveByName("jitter_count")
		return app.Save(collection)
	})
}
