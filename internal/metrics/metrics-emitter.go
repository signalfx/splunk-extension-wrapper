package metrics

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/signalfx/golib/v3/sfxclient"
	"github.com/splunk/lambda-extension/internal/config"
	"log"
	"os"
)

const awsExecutionEnv = "AWS_EXECUTION_ENV"

type MetricEmitter struct {
	config    *config.Configuration
	scheduler *sfxclient.Scheduler
	started   bool

	functionName    string
	functionVersion string

	invocationsCounters map[string]*invocationsCounter

	environmentMetrics
}

func New() *MetricEmitter {
	configuration := config.New()

	scheduler := sfxclient.NewScheduler()
	scheduler.Sink.(*sfxclient.HTTPSink).DatapointEndpoint = configuration.IngestURL
	scheduler.Sink.(*sfxclient.HTTPSink).AuthToken = configuration.Token
	scheduler.ReportingDelay(configuration.ReportingDelay)
	scheduler.ReportingTimeout(configuration.ReportingTimeout)

	emitter := &MetricEmitter{
		config:    &configuration,
		scheduler: scheduler,

		invocationsCounters: make(map[string]*invocationsCounter),

		environmentMetrics: newEnvironmentMetrics(),
	}

	scheduler.AddCallback(&emitter.environmentMetrics)

	emitter.environmentMetrics.markStart()

	return emitter
}

func (emitter *MetricEmitter) Invoked(functionArn string) {
	if counter, found := emitter.invocationsCounters[functionArn]; found {
		counter.invoked()
	} else {
		emitter.registerCounter(functionArn)
	}

	if !emitter.started {
		dims := emitter.dims(functionArn)
		delete(dims, dimQualifier) // the env metrics are only related to the function version
		emitter.scheduler.DefaultDimensions(dims)
		go emitter.scheduler.Schedule(context.Background())
		emitter.started = true
	}
}

func (emitter *MetricEmitter) registerCounter(functionArn string) {
	counter := &invocationsCounter{}
	counter.invoked()

	emitter.invocationsCounters[functionArn] = counter

	emitter.scheduler.GroupedDefaultDimensions(functionArn, emitter.dims(functionArn))
	emitter.scheduler.AddGroupedCallback(functionArn, counter)
}

func (emitter *MetricEmitter) SetFunction(functionName, functionVersion string) {
	emitter.functionName = functionName
	emitter.functionVersion = functionVersion
}

func (emitter *MetricEmitter) Shutdown(reason string) {
	if !emitter.started {
		log.Printf("closing emitter that wasn't started")
	}

	emitter.environmentMetrics.markEnd(reason)

	if err := emitter.scheduler.ReportOnce(context.Background()); err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("failed to shutdown emitter: %v\n", err)
	}
}

func (emitter MetricEmitter) buildAWSUniqueId(functionArn arn.ARN) string {
	return fmt.Sprintf("lambda_%s_%s_%s", emitter.functionName, functionArn.Region, functionArn.AccountID)
}

func (emitter MetricEmitter) arnWithVersion(parsedArn arn.ARN) string {
	resource := resourceFromArn(parsedArn)

	if emitter.functionVersion != "$LATEST" {
		resource.qualifier = emitter.functionVersion
	} else {
		resource.qualifier = emptyQualifier
	}

	parsedArn.Resource = resource.String()

	return parsedArn.String()
}
