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
