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

// This is purely a throwaway prototype!

package logs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
)

func inboundHttp(w http.ResponseWriter, req *http.Request) {
	fmt.Println("JB LOGS Got inbound http")
	if req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			fmt.Println("JB LOGS read err")
			return
		}
		defer req.Body.Close()
		fmt.Println(string(body))
	}
	w.WriteHeader(200)
}

func startServer() {
	http.HandleFunc("/", inboundHttp)
	go http.ListenAndServe(":1518", nil)
}

type LogDestination struct {
	Protocol	string		`json:"protocol"`
	URI		string
}

type LogSubscribeRequest struct {
	Destination	LogDestination	`json:"destination"`
	Types		[]string	`json:"types"`
	SchemaVersion	string		`json:"schemaVersion"`
}

func SubscribeToLogs(apiExtensionId string) {
	fmt.Println("JB logs init")
	startServer()

	_ = otlpexporter.NewFactory()

	apiHost := os.Getenv("AWS_LAMBDA_RUNTIME_API")

        subscribeUrl := fmt.Sprintf("http://%v/2022-07-01/telemetry", apiHost)

	sub, err := json.Marshal(LogSubscribeRequest{
		Destination: LogDestination{
			Protocol: "HTTP",
			URI: "http://sandbox.localdomain:1518/"},
		Types: []string{"function"},
		SchemaVersion: "2022-07-01"})

	if err != nil {
		fmt.Println("JB logs no json")
		return
	}
	fmt.Println(string(sub))

	req, err := http.NewRequest(http.MethodPut, subscribeUrl, bytes.NewBuffer(sub))

	if err != nil {
		fmt.Println("JB logs new req err")
		return
	}

	req.Header.Set("Lambda-Extension-Identifier", apiExtensionId)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		fmt.Println("JB logs client err")
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	body := string(bodyBytes)
	fmt.Println("JB logs got body!")
	fmt.Println(body)

	if resp.StatusCode != http.StatusOK {
		fmt.Println("JB logs not OK")
		fmt.Println(resp.StatusCode)
		return
	}
}

