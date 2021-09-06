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
	"testing"
)

func TestReplacingVersionInResource(t *testing.T) {
	lambdaArn, _ := arn.Parse("arn:aws:lambda:aws-region:acct-id:function:helloworld:42")

	resource := resourceFromArn(lambdaArn)
	resource.qualifier = "10"

	expected := "function:helloworld:10"
	actual := resource.String()

	if expected != actual {
		t.Errorf("Expected `%v`, got `%v`", expected, actual)
	}
}
