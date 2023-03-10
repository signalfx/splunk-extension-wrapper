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

package extensionapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/splunk/lambda-extension/internal/config"
	"github.com/splunk/lambda-extension/internal/shutdown"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	invokeType   = "INVOKE"
	shutdownType = "SHUTDOWN"
)

type registerResponse struct {
	FunctionName    string
	FunctionVersion string
	Handler         string
}

type Event struct {
	EventType          string
	DeadlineMs         int64
	RequestId          string
	InvokedFunctionArn string
	ShutdownReason     string
}

type RegisteredApi struct {
	ExtensionName string

	extensionId string

	registerResponse
}

func Register(name string, configuration *config.Configuration) (*RegisteredApi, shutdown.Condition) {
	log.Println("Registering...")

	rb, err := json.Marshal(map[string][]string{
		"events": {invokeType, shutdownType}})

	if err != nil {
		return nil, shutdown.Api(fmt.Sprintf("can't marshall body: %v", err))
	}

	transportCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: configuration.InsecureSkipHTTPSVerify},
	}
	client := &http.Client{Transport: transportCfg}

	req, err := http.NewRequest(http.MethodPost, endpoints.register, bytes.NewBuffer(rb))

	if err != nil {
		return nil, shutdown.Api(fmt.Sprintf("can't create http request: %v", err))
	}

	req.Header.Set("Lambda-Extension-Name", name)

	resp, err := client.Do(req)

	if err != nil {
		return nil, shutdown.Api(fmt.Sprintf("can't register: %v", err))
	}
	defer resp.Body.Close()

	log.Printf("Register status code: %v\n", resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, shutdown.Api(fmt.Sprintf("can't read body: %v", err))
	}

	body := string(bodyBytes)

	log.Printf("Register response: %v\n", body)

	if resp.StatusCode != http.StatusOK {
		return nil, shutdown.Api("failed to register, API returned: " + resp.Status)
	}

	id, has := resp.Header["Lambda-Extension-Identifier"]

	if !has || len(id) != 1 {
		return nil, shutdown.Api(fmt.Sprintf("Lambda-Extension-Identifier header missing or ambiguous: %v", id))
	}

	regResponse := &registerResponse{}
	err = json.Unmarshal(bodyBytes, regResponse)

	if err != nil {
		return nil, shutdown.Api(fmt.Sprintf("unknown format of a register response: %v", err))
	}

	log.Println("Registering [DONE]")

	log.Printf("Unmarshalled register response: %v\n", *regResponse)

	return &RegisteredApi{
		ExtensionName:    name,
		extensionId:      id[0],
		registerResponse: *regResponse}, nil
}

func (api RegisteredApi) NextEvent() (*Event, shutdown.Condition) {
	log.Println("Waiting for event")

	req, err := http.NewRequest(http.MethodGet, endpoints.next, nil)

	if err != nil {
		return nil, shutdown.Api(fmt.Sprintf("can't create http request: %v", err))
	}

	req.Header.Set("Lambda-Extension-Identifier", api.extensionId)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, shutdown.Api(fmt.Sprintf("can't get next event: %v", err))
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, shutdown.Api(fmt.Sprintf("can't read body: %v", err))
	}

	body := string(bodyBytes)

	log.Printf("Received event: %v\n", body)

	if resp.StatusCode != http.StatusOK {
		return nil, shutdown.Api("failed to get the next event, API returned: " + resp.Status)
	}

	nextResp := &Event{}
	err = json.Unmarshal(bodyBytes, nextResp)
	if err != nil {
		return nil, shutdown.Api(fmt.Sprintf("unknown format of an event: %v", err))
	}

	log.Printf("Unmarshaled event: %v\n", *nextResp)

	if nextResp.EventType == shutdownType {
		return nil, shutdown.Reason(nextResp.ShutdownReason)
	}

	return nextResp, nil
}
