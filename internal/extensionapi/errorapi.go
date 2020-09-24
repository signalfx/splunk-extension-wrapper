package extensionapi

import (
	"fmt"
	"log"
	"net/http"
)

func (api RegisteredApi) InitError(errorType string) error {
	log.Println("Reporting an init error")

	err := api.reportError(endpoints.initError, errorType)

	return wrapIfNotNull("failed to report an init error", err)
}

func (api RegisteredApi) ExitError(errorType string) error {
	log.Println("Reporting an exit error")

	err := api.reportError(endpoints.exitError, errorType)

	log.Printf("Reporting an exit error [DONE]")

	return wrapIfNotNull("failed to report an exit error", err)
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

func wrapIfNotNull(context string, err error) error {
	if err != nil {
		return fmt.Errorf(context+": %w", err)
	}
	return nil
}
