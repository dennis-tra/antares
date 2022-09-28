package metrics

import (
	"fmt"
	"net/http"

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

func RegisterMetrics() error {
	if err := view.Register(DefaultStartViews...); err != nil {
		return errors.Wrap(err, "register antares default views")
	}
	return nil
}

func ListenAndServe(host string, port int) error {
	// Register default Go and process metrics
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Initialize new exporter instance
	pe, err := ocprom.NewExporter(ocprom.Options{
		Namespace: "antares",
		Registry:  registry,
	})
	if err != nil {
		return errors.Wrap(err, "new prometheus exporter")
	}

	// Enable ocsql metrics with OpenCensus
	ocsql.RegisterAllViews()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		if err := http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), mux); err != nil {
			log.Fatalf("Failed to run Prometheus /metrics endpoint: %v", err)
		}
	}()
	return nil
}

// Keys
var (
	KeyTargetName, _ = tag.NewKey("target_name")
	KeyTargetType, _ = tag.NewKey("target_type")
)

// Measures
var (
	ProbeCount = stats.Int64("probe_count", "Number probes performed", stats.UnitDimensionless)
	TrackCount = stats.Int64("track_count", "Number tracked peers", stats.UnitDimensionless)
)

// Views
var (
	ProbeCountView = &view.View{
		Measure:     ProbeCount,
		TagKeys:     []tag.Key{KeyTargetName, KeyTargetType},
		Aggregation: view.Count(),
	}
	TrackCountView = &view.View{
		Measure:     TrackCount,
		TagKeys:     []tag.Key{KeyTargetName, KeyTargetType},
		Aggregation: view.Count(),
	}
)

// DefaultStartViews with all views in it.
var DefaultStartViews = []*view.View{
	ProbeCountView,
	TrackCountView,
}
