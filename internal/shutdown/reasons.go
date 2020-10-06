package shutdown

const (
	internalError = "internal"
	apiError      = "api"
	metricError   = "metric"
)

type Condition interface {
	Reason() string
	Message() string
	IsError() bool
}

type simple struct {
	reason  string
	message string
	error   bool
}

func newWithError(message, reason string) *simple {
	return &simple{message: message, reason: reason, error: true}
}

func (s simple) Reason() string {
	return s.reason
}

func (s simple) Message() string {
	return s.message
}

func (s simple) IsError() bool {
	return s.error
}

func Api(message string) Condition {
	return newWithError(message, apiError)
}

func Internal(message string) Condition {
	return newWithError(message, internalError)
}

func Metric(message string) Condition {
	return newWithError(message, metricError)
}

func Reason(reason string) Condition {
	return simple{reason: reason}
}
