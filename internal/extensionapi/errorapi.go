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

	req.Header.Set("Lambda-Extension-Identifier", api.ExtensionId)
	req.Header.Set("Lambda-Extension-Function-Error-Type", errorType)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Printf("failed to send request: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Println("API returned: ", resp.Status)
}
