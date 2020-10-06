package extensionapi

import (
	"fmt"
	"os"
)

type apiEndpoints struct {
	register, next, initError, exitError string
}

var apiHost = os.Getenv("AWS_LAMBDA_RUNTIME_API")

var endpoints = apiEndpoints{
	register:  fmt.Sprintf("http://%v/2020-01-01/extension/register", apiHost),
	next:      fmt.Sprintf("http://%v/2020-01-01/extension/event/next", apiHost),
	initError: fmt.Sprintf("http://%v/2020-01-01/extension/init/error", apiHost),
	exitError: fmt.Sprintf("http://%v/2020-01-01/extension/exit/error", apiHost)}
