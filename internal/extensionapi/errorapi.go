package extensionapi

import (
	"log"
	"net/http"
)

// after calling any of these functions, the extension should exit immediately

func (api RegisteredApi) InitError(errorType string) {
	log.Println("Reporting an init error: ", errorType)

	api.reportError(endpoints.initError, errorType)
}

func (api RegisteredApi) ExitError(errorType string) {
	log.Println("Reporting an exit error: ", errorType)

	api.reportError(endpoints.exitError, errorType)
}

func (api RegisteredApi) reportError(endpoint, errorType string) {
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)

	if err != nil {
		log.Printf("can't create http request: %v", err)
		return
	}

	req.Header.Set("Lambda-Extension-Identifier", api.extensionId)
	req.Header.Set("Lambda-Extension-Function-Error-Type", errorType)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Printf("failed to send request: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Println("API returned: ", resp.Status)
}
