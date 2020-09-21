package metrics

import (
	"context"
	"github.com/signalfx/golib/v3/sfxclient"
	"github.com/splunk/lambda-extension/internal/config"
	"log"
)

type MetricEmitter struct {
	config    *config.Configuration
	scheduler *sfxclient.Scheduler

	started   bool
	initError chan error

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

		initError: make(chan error),

		metrics: newMetrics(),
	}

	scheduler.AddCallback(&emitter.metrics)

	emitter.metrics.markStart()

	return emitter
}

func (emitter MetricEmitter) AlreadyStarted() bool {
	return emitter.started
}

func (emitter *MetricEmitter) StartScheduler() {
	go func() {
		log.Printf("going to report once\n")
		err := emitter.scheduler.ReportOnce(context.Background())
		log.Printf("datapoints reported once with error: %v\n", err)
		if err != nil {
			emitter.initError <- err
			return
		}
		log.Printf("starting scheduler")
		emitter.scheduler.Schedule(context.Background())
	}()
	emitter.started = true
}

func (emitter *MetricEmitter) Dimensions(name, version, id string) {
	emitter.scheduler.DefaultDimensions(map[string]string{
		dimFunctionName:    name,
		dimFunctionVersion: version,
		dimAwsUniqueId:     id,
	})
}

func (emitter *MetricEmitter) Shutdown(reason string) error {
	emitter.metrics.markEnd(reason)

	return emitter.scheduler.ReportOnce(context.Background())
}

func (emitter MetricEmitter) InitError() error {
	select {
	case err := <-emitter.initError:
		return err
	default:
		return nil
	}
}
