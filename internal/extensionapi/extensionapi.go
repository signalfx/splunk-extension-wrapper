package extensionapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/arn"
	"io/ioutil"
	"log"
	"net/http"
)

const invoke = "INVOKE"
const shutdown = "SHUTDOWN"

type registerResponse struct {
	FunctionName    string
	FunctionVersion string
	Handler         string
}

type NextResponse struct {
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

func Register(name string) (*RegisteredApi, error) {
	log.Println("Registering...")

	rb, err := json.Marshal(map[string][]string{
		"events": {"INVOKE", "SHUTDOWN"}})

	if err != nil {
		return nil, fmt.Errorf("can't marshall body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoints.register, bytes.NewBuffer(rb))

	if err != nil {
		return nil, fmt.Errorf("can't create http request: %v", err)
	}

	req.Header.Set("Lambda-Extension-Name", name)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("can't register: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Register status code: %v\n", resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read body: %v", err)
	}

	body := string(bodyBytes)

	log.Printf("Register response: %v\n", body)

	if resp.StatusCode != http.StatusOK {
		return nil, ApiError("failed to register, API returned: " + resp.Status)
	}

	id, has := resp.Header["Lambda-Extension-Identifier"]

	if !has || len(id) > 1 {
		return nil, fmt.Errorf("Lambda-Extension-Identifier header missing in response")
	}

	regResponse := &registerResponse{}
	err = json.Unmarshal(bodyBytes, regResponse)

	if err != nil {
		return nil, fmt.Errorf("unknown format of a register response")
	}

	log.Println("Registering [DONE]")

	log.Printf("Unmarshalled register response: %v\n", *regResponse)

	return &RegisteredApi{
		ExtensionName:    name,
		extensionId:      id[0],
		registerResponse: *regResponse}, nil
}

func (api RegisteredApi) Next() (*NextResponse, error) {
	log.Println("Waiting for event")

	req, err := http.NewRequest(http.MethodGet, endpoints.next, nil)

	if err != nil {
		return nil, fmt.Errorf("can't create http request: %v", err)
	}

	req.Header.Set("Lambda-Extension-Identifier", api.extensionId)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("can't get next event: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read body: %v", err)
	}

	body := string(bodyBytes)

	log.Printf("Received event: %v\n", body)

	if resp.StatusCode != http.StatusOK {
		return nil, ApiError("failed to get the next event, API returned: " + resp.Status)
	}

	nextResp := &NextResponse{}
	err = json.Unmarshal(bodyBytes, nextResp)
	if err != nil {
		return nil, fmt.Errorf("unknown format of an event")
	}

	log.Printf("Unmarshaled event: %v\n", *nextResp)

	return nextResp, nil
}

func (next NextResponse) ParseArn() (string, string) {
	arn, err := arn.Parse(next.InvokedFunctionArn)

	if err != nil {
		log.Panicf("can't parse ARN: %v\n", next.InvokedFunctionArn)
	}

	return arn.Region, arn.AccountID
}

func (next NextResponse) AWSUniqueId(functionName string) string {
	region, account := next.ParseArn()

	return fmt.Sprintf("lambda_%s_%s_%s", functionName, region, account)
}

func (next NextResponse) IsShutdown() bool {
	return next.EventType == shutdown
}

func (next NextResponse) GetShutdownReason() string {
	return next.ShutdownReason
}
