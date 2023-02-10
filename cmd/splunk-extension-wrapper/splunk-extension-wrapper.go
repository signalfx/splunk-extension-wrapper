// Copyright Splunk Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/splunk/lambda-extension/internal/config"
	"github.com/splunk/lambda-extension/internal/extensionapi"
	"github.com/splunk/lambda-extension/internal/metrics"
	"github.com/splunk/lambda-extension/internal/ossignal"
	"github.com/splunk/lambda-extension/internal/shutdown"
	"bufio"
	"fmt"
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
	configuration := config.New()

	initLogging(&configuration)

	ossignal.Watch()

	m := metrics.New()

	shutdownCondition := registerApiAndStartMainLoop(m, &configuration)

	if shutdownCondition.IsError() {
		log.SetOutput(os.Stderr)
	}

	log.Println("shutdown reason:", shutdownCondition.Reason())
	log.Println("shutdown message:", shutdownCondition.Message())

	m.Shutdown(shutdownCondition)
}

func registerApiAndStartMainLoop(m *metrics.MetricEmitter, configuration *config.Configuration) (sc shutdown.Condition) {
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

	api, sc = extensionapi.Register(extensionName(), configuration)

	if sc == nil {
		sc = mainLoop(api, m, configuration)
	}

	if sc != nil && sc.IsError() && api != nil {
		api.ExitError(sc.Reason())
	}

	return
}

func mainLoop(api *extensionapi.RegisteredApi, m *metrics.MetricEmitter, configuration *config.Configuration) (sc shutdown.Condition) {
	m.SetFunction(api.FunctionName, api.FunctionVersion)

	var event *extensionapi.Event
	event, sc = api.NextEvent()

	for sc == nil {
		sc = m.Invoked(event.InvokedFunctionArn, configuration.SplunkFailFast)
		if sc == nil {
			event, sc = api.NextEvent()
		}
	}

	return
}

func initLogging(configuration *config.Configuration) {
	en := extensionName()
	log.SetPrefix("[" + en + "] ")
	log.SetFlags(log.Lmsgprefix)

	log.Printf("%v, version: %v", extensionName(), gitVersion)

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
