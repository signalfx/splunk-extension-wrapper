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
