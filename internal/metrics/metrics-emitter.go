package metrics

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/signalfx/golib/v3/sfxclient"
	"github.com/splunk/lambda-extension/internal/config"
	"github.com/splunk/lambda-extension/internal/shutdown"
	"github.com/splunk/lambda-extension/internal/util"
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

	arnToCounter map[string]*invocationsCounter

	ctx context.Context

	sendOutTicker util.Ticker

	environmentMetrics
}

func New() *MetricEmitter {
	configuration := config.New()

	scheduler := sfxclient.NewScheduler()
	scheduler.Sink.(*sfxclient.HTTPSink).DatapointEndpoint = configuration.SplunkIngestUrl + "/v2/datapoint"
	scheduler.Sink.(*sfxclient.HTTPSink).AuthToken = configuration.SplunkToken
	scheduler.Sink.(*sfxclient.HTTPSink).Client.Timeout = configuration.ReportingTimeout
	scheduler.ReportingTimeout(configuration.ReportingTimeout)

	emitter := &MetricEmitter{
		config:    &configuration,
		scheduler: scheduler,

		arnToCounter: make(map[string]*invocationsCounter),

		ctx: context.Background(),

		sendOutTicker: util.NewTicker(configuration),
	}

	if configuration.HttpTracing {
		emitter.ctx = util.WithClientTracing(emitter.ctx)
	}

	scheduler.AddCallback(&emitter.environmentMetrics)

	emitter.environmentMetrics.markStart()

	return emitter
}

func (emitter *MetricEmitter) Invoked(functionArn string) shutdown.Condition {
	if counter, found := emitter.arnToCounter[functionArn]; found {
		counter.invoked()
	} else {
		emitter.registerCounter(functionArn)
	}

	if !emitter.started {
		emitter.markFirstInvocation()
		dims := emitter.dims(functionArn)
		delete(dims, dimQualifier) // the env metrics are only related to the function version
		emitter.scheduler.DefaultDimensions(dims)
		emitter.started = true
	}

	return emitter.tryToSendOut()
}

func (emitter *MetricEmitter) SetFunction(functionName, functionVersion string) {
	emitter.functionName = functionName
	emitter.functionVersion = functionVersion
}

func (emitter *MetricEmitter) Shutdown(condition shutdown.Condition) {
	if !emitter.started {
		log.Printf("closing emitter that wasn't started")
	}

	emitter.environmentMetrics.markEnd(condition.Reason())

	if err := emitter.scheduler.ReportOnce(emitter.ctx); err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("failed to report metrics on shutdown: %v\n", err)
	}
}

func (emitter MetricEmitter) buildAWSUniqueId(functionArn arn.ARN) string {
	return fmt.Sprintf("lambda_%s_%s_%s", emitter.functionName, functionArn.Region, functionArn.AccountID)
}

func (emitter MetricEmitter) arnWithVersion(parsedArn arn.ARN) string {
	resource := resourceFromArn(parsedArn)

	resource.qualifier = emitter.functionVersion
	parsedArn.Resource = resource.String()

	return parsedArn.String()
}

func (emitter *MetricEmitter) tryToSendOut() shutdown.Condition {
	if !emitter.sendOutTicker.Tick() {
		return nil
	}
	log.Println("sending metrics")
	err := emitter.scheduler.ReportOnce(emitter.ctx)
	if err == nil {
		return nil
	}

	return shutdown.Metric(fmt.Sprintf("failed to send metrics: %v", err))
}

func (emitter *MetricEmitter) registerCounter(functionArn string) {
	counter := &invocationsCounter{}
	counter.invoked()

	emitter.arnToCounter[functionArn] = counter

	emitter.scheduler.GroupedDefaultDimensions(functionArn, emitter.dims(functionArn))
	emitter.scheduler.AddGroupedCallback(functionArn, counter)
}
