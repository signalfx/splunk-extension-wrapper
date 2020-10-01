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

	metrics
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

		metrics: newMetrics(),
	}

	scheduler.AddCallback(&emitter.metrics)

	emitter.metrics.markStart()

	return emitter
}

func (emitter *MetricEmitter) StartScheduler() {
	go emitter.scheduler.Schedule(context.Background())
}

func (emitter *MetricEmitter) SetDefaultDimensions(functionArn, functionName, functionVersion string) {
	parsedArn, err := arn.Parse(functionArn)

	if err != nil {
		log.Panicf("can't parse ARN: %v\n", functionArn)
	}

	emitter.scheduler.DefaultDimensions(map[string]string{
		dimRegion:          parsedArn.Region,
		dimAccountId:       parsedArn.AccountID,
		dimFunctionName:    functionName,
		dimFunctionVersion: functionVersion,
		dimQualifier:       resourceFromArn(parsedArn).qualifier,
		dimArn:             functionArn,
		dimRuntime:         os.Getenv(awsExecutionEnv),
		dimAwsUniqueId:     buildAWSUniqueId(parsedArn, functionName),
	})
}

func (emitter *MetricEmitter) Shutdown(reason string) {
	emitter.metrics.markEnd(reason)

	if err := emitter.scheduler.ReportOnce(context.Background()); err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("failed to shutdown emitter: %v\n", err)
	}
}

func buildAWSUniqueId(functionArn arn.ARN, functionName string) string {
	return fmt.Sprintf("lambda_%s_%s_%s", functionName, functionArn.Region, functionArn.AccountID)
}
