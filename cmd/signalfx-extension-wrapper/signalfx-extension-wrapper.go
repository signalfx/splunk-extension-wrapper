package main

import (
	"github.com/splunk/lambda-extension/internal/config"
	"github.com/splunk/lambda-extension/internal/extensionapi"
	"github.com/splunk/lambda-extension/internal/metrics"
	"github.com/splunk/lambda-extension/internal/ossignal"
	"io/ioutil"
	"log"
	"os"
	"path"
)

// the correct value is set by the go linker (it's done during build using "ldflags")
var gitVersion string

func main() {
	initLogging()

	ossignal.Watch()

	var api *extensionapi.RegisteredApi
	m := metrics.New()

	defer func() {
		if r := recover(); r != nil {
			log.SetOutput(os.Stderr)
			log.Printf("panic condition: %v\n", r)
			if api != nil {
				api.ExitError("Internal error")
			}
			m.Shutdown("Internal error")
			os.Exit(1)
		}
	}()

	api, apiErr := extensionapi.Register(extensionName())

	if apiErr == nil {
		m.SetFunction(api.FunctionName, api.FunctionVersion)
		event, apiErr := api.NextEvent()
		if apiErr == nil {
			for apiErr == nil && !event.IsShutdown() {
				m.Invoked(event.InvokedFunctionArn)
				event, apiErr = api.NextEvent()
			}

			if event.IsShutdown() {
				m.Shutdown(event.ShutdownReason)
			}
		}
	}

	if apiErr != nil {
		reason := toReason(apiErr)
		if api != nil {
			api.ExitError(reason)
		}
		m.Shutdown(reason)
	}
}

func initLogging() {
	en := extensionName()
	log.SetPrefix("[" + en + "] ")
	log.SetFlags(log.Lmsgprefix)

	log.Printf("%v, version: %v", extensionName(), gitVersion)
	log.Printf("lambda region: %v", os.Getenv("AWS_REGION"))
	log.Printf("lambda runtime: %v", os.Getenv("AWS_EXECUTION_ENV"))

	configuration := config.New()

	if !configuration.Verbose {
		log.SetOutput(ioutil.Discard)
	}

	log.Println(configuration)
}

func extensionName() string {
	return path.Base(os.Args[0])
}

func toReason(err error) string {
	if _, ok := err.(*extensionapi.ApiError); ok {
		return "API error"
	}
	return "Internal error"
}
