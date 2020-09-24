package metrics

import (
	"context"
	"github.com/signalfx/golib/v3/sfxclient"
	"github.com/splunk/lambda-extension/internal/config"
	"log"
	"os"
)

type MetricEmitter struct {
	config    *config.Configuration
	scheduler *sfxclient.Scheduler

	metrics
}

func New() *MetricEmitter {
	configuration := config.New()

	scheduler := sfxclient.NewScheduler()
	scheduler.Sink.(*sfxclient.HTTPSink).DatapointEndpoint = configuration.IngestURL
	scheduler.Sink.(*sfxclient.HTTPSink).AuthToken = configuration.Token
	scheduler.ReportingDelay(configuration.ReportingDelay)

	emitter := &MetricEmitter{
		config:    &configuration,
		scheduler: scheduler,

		metrics: newMetrics(),
	}

	scheduler.AddCallback(&emitter.metrics)

	emitter.metrics.markStart()

	return emitter
}

func (emitter *MetricEmitter) StartScheduler() {
	go emitter.scheduler.Schedule(context.Background())
}

func (emitter *MetricEmitter) SetDefaultDimensions(name, version, id string) {
	emitter.scheduler.DefaultDimensions(map[string]string{
		dimFunctionName:    name,
		dimFunctionVersion: version,
		dimAwsUniqueId:     id,
	})
}

func (emitter *MetricEmitter) Shutdown(reason string) {
	emitter.metrics.markEnd(reason)

	if err := emitter.scheduler.ReportOnce(context.Background()); err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("failed to shutdown emitter: %v\n", err)
	}
}
