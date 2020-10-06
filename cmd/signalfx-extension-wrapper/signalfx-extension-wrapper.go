package main

import (
	"bufio"
	"fmt"
	"github.com/splunk/lambda-extension/internal/config"
	"github.com/splunk/lambda-extension/internal/extensionapi"
	"github.com/splunk/lambda-extension/internal/metrics"
	"github.com/splunk/lambda-extension/internal/ossignal"
	"github.com/splunk/lambda-extension/internal/shutdown"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
)

// the correct value is set by the go linker (it's done during build using "ldflags")
var gitVersion string

func main() {
	initLogging()

	ossignal.Watch()

	m := metrics.New()

	shutdownCondition := registerApiAndStartMainLoop(m)

	if shutdownCondition.IsError() {
		log.SetOutput(os.Stderr)
	}

	log.Println("shutdown reason:", shutdownCondition.Reason())
	log.Println("shutdown message:", shutdownCondition.Message())

	m.Shutdown(shutdownCondition)
}

func registerApiAndStartMainLoop(m *metrics.MetricEmitter) (sc shutdown.Condition) {
	var api *extensionapi.RegisteredApi

	defer func() {
		if r := recover(); r != nil {
			log.SetOutput(os.Stderr)
			sc = shutdown.Internal(fmt.Sprintf("%v", r))
			if api != nil {
				api.ExitError(sc.Reason())
			}
		}
	}()

	api, sc = extensionapi.Register(extensionName())

	if sc == nil {
		sc = mainLoop(api, m)
	}

	if sc != nil && api != nil {
		api.ExitError(sc.Reason())
	}

	return
}

func mainLoop(api *extensionapi.RegisteredApi, m *metrics.MetricEmitter) (sc shutdown.Condition) {
	m.SetFunction(api.FunctionName, api.FunctionVersion)

	var event *extensionapi.Event
	event, sc = api.NextEvent()

	for sc == nil {
		sc = m.Invoked(event.InvokedFunctionArn)
		if sc == nil {
			event, sc = api.NextEvent()
		}
	}

	return
}

func initLogging() {
	en := extensionName()
	log.SetPrefix("[" + en + "] ")
	log.SetFlags(log.Lmsgprefix)

	log.Printf("%v, version: %v", extensionName(), gitVersion)

	configuration := config.New()

	if !configuration.Verbose {
		log.SetOutput(ioutil.Discard)
	}

	log.Printf("lambda region: %v", os.Getenv("AWS_REGION"))
	log.Printf("lambda runtime: %v", os.Getenv("AWS_EXECUTION_ENV"))

	log.Println("GOMAXPROCS", runtime.GOMAXPROCS(0))
	log.Println("NumCPU", runtime.NumCPU())
	log.Println("goroutines on start", runtime.NumGoroutine())

	scanner := bufio.NewScanner(strings.NewReader(configuration.String()))
	for scanner.Scan() {
		log.Print(scanner.Text())
	}
}

func extensionName() string {
	return path.Base(os.Args[0])
}
