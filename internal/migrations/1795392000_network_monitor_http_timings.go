package migrations

import (
	"ganigeorgiev.com/pocketbase/pocketbase/core"
	m "ganigeorgiev.com/pocketbase/pocketbase/migrations"
)

// httpTimingFields are sums of HTTP phase durations over successful checks, in
// microseconds. Dividing by success_count gives the average, like res_sum.
var httpTimingFields = []struct{ id, name, help string }{
	{"nms_dns_sum", "dns_sum", "Sum of HTTP DNS lookup times in microseconds"},
	{"nms_connect_sum", "connect_sum", "Sum of HTTP TCP connect times in microseconds"},
	{"nms_tls_sum", "tls_sum", "Sum of HTTP TLS handshake times in microseconds"},
	{"nms_server_sum", "server_sum", "Sum of HTTP times to first response byte in microseconds"},
}

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("network_monitor_stats")
		if err != nil {
			return err
		}
		for _, field := range httpTimingFields {
			collection.Fields.Add(&core.NumberField{
				Id:      field.id,
				Name:    field.name,
				Help:    field.help,
				Min:     new(float64),
				OnlyInt: true,
			})
		}
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("network_monitor_stats")
		if err != nil {
			return err
		}
		for _, field := range httpTimingFields {
			collection.Fields.RemoveByName(field.name)
		}
		return app.Save(collection)
	})
}
