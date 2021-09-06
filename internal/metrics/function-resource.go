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

package metrics

import (
	"github.com/aws/aws-sdk-go/aws/arn"
	"log"
	"strings"
)

const delimiter = ":"
const emptyQualifier = ""

type functionResource struct {
	kind, id, qualifier string
}

func resourceFromArn(arn arn.ARN) functionResource {
	split := strings.Split(arn.Resource, delimiter)

	if len(split) < 2 {
		log.Panicf("can't parse ARN: %v (invalid resource)\n", arn)
	}

	qualifier := emptyQualifier
	if len(split) > 2 {
		qualifier = split[2]
	}

	return functionResource{
		kind:      split[0],
		id:        split[1],
		qualifier: qualifier,
	}
}

func (resource functionResource) String() (str string) {
	str = resource.kind + delimiter + resource.id

	if resource.qualifier != emptyQualifier {
		str += delimiter + resource.qualifier
	}

	return
}
