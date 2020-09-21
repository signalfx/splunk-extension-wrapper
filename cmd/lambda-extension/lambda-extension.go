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

func main() {
	initLogging()

	ossignal.Watch()

	var apiErr error
	var initError error
	var api *extensionapi.RegisteredApi
	var shutdownReason string

	m := metrics.New()

	defer func() {
		log.SetOutput(os.Stderr)
		if r := recover(); r != nil || apiErr != nil {
			log.Printf("API error %v\n", apiErr)
			log.Printf("panic condition: %v\n", r)
			shutdownReason = toReason(apiErr)
			if api != nil {
				api.ExitError(shutdownReason)
			}
		}

		if initError != nil {
			log.Printf("init error: %v\n", initError)
			if api != nil {
				api.InitError("Fatal")
			}
		}

		if err := m.Shutdown(shutdownReason); err != nil {
			log.Printf("failed to shutdown emitter: %v\n", err)
		}

		if config.New().Verbose {
			log.Printf("Exiting...\n")
		}
		os.Exit(1)
	}()

	api, apiErr = extensionapi.Register(extensionName())

	for apiErr == nil {
		initError = m.InitError()
		if initError != nil {
			break
		}

		var nextResponse *extensionapi.NextResponse
		nextResponse, apiErr = api.Next()
		if apiErr != nil {
			continue
		}

		if !m.AlreadyStarted() {
			m.Dimensions(api.FunctionName, api.FunctionVersion, nextResponse.AWSUniqueId(api.FunctionName))
			m.StartScheduler()
		}

		if nextResponse.IsShutdown() {
			shutdownReason = nextResponse.GetShutdownReason()
			break
		}

		m.Invoked()
	}
}

func initLogging() {
	en := extensionName()
	log.SetPrefix("[" + en + "] ")
	log.SetFlags(log.Lmsgprefix)

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
	if err != nil {
		if _, ok := err.(*extensionapi.ApiError); ok {
			return "API error"
		}
	}
	return "Internal error"
}
