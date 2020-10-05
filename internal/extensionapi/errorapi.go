package extensionapi

import (
	"fmt"
	"log"
	"net/http"
)

// after calling any of these functions, the extension should exit immediately

func (api RegisteredApi) InitError(errorType string) {
	log.Println("Reporting an init error")

	err := api.reportError(endpoints.initError, errorType)

	log.Println("Reporting an init error is done with error: ", err)
}

func (api RegisteredApi) ExitError(errorType string) {
	log.Println("Reporting an exit error")

	err := api.reportError(endpoints.exitError, errorType)

	log.Println("Reporting an exit error is done with error: ", err)
}

func (api RegisteredApi) reportError(endpoint, errorType string) error {
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)

	if err != nil {
		return fmt.Errorf("can't create http request: %v", err)
	}

	req.Header.Set("Lambda-Extension-Identifier", api.extensionId)
	req.Header.Set("Lambda-Extension-Function-Error-Type", errorType)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ApiError("API returned: " + resp.Status)
	}

	return nil
}
