package metrics

import (
	"github.com/aws/aws-sdk-go/aws/arn"
	"log"
	"strings"
)

type functionResource struct {
	kind, id, qualifier string
}

func resourceFromArn(arn arn.ARN) functionResource {
	split := strings.Split(arn.Resource, ":")

	if len(split) < 2 {
		log.Panicf("can't parse ARN: %v (invalid resource)\n", arn)
	}

	qualifier := ""
	if len(split) > 2 {
		qualifier = split[2]
	}

	return functionResource{
		kind:      split[0],
		id:        split[1],
		qualifier: qualifier,
	}
}
