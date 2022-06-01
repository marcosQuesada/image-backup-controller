package registry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	existsCalls = promauto.NewCounter(prometheus.CounterOpts{
		Name: "backup_registry_backup_exists_total",
		Help: "The total number of provider processed exists calls",
	})

	existsErroredCalls = promauto.NewCounter(prometheus.CounterOpts{
		Name: "backup_registry_backup_error_exists_total",
		Help: "The total number of provider processed exists calls with error result",
	})

	existsDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "backup_registry_backup_exists_duration",
		Help: "The duration of provider processed exists calls",
	})

	backupCalls = promauto.NewCounter(prometheus.CounterOpts{
		Name: "backup_registry_backup_execution_total",
		Help: "The total number of provider backup request calls",
	})

	backupErroredCalls = promauto.NewCounter(prometheus.CounterOpts{
		Name: "backup_registry_backup_error_execution_total",
		Help: "The total number of provider backup request calls with error result",
	})

	backupDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "backup_registry_backup_execution_duration",
		Help: "The total number of provider backup duration calls",
	})
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(existsCalls, existsDuration, backupCalls, backupErroredCalls, backupDuration)
}
