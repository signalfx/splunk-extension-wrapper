package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultIngestURL = "https://ingest.signalfx.com/v2/datapoint"
const defaultToken = ""
const defaultFastIngest = true
const defaultReportingDuration = time.Duration(15) * time.Second
const defaultReportingTimeout = time.Duration(5) * time.Second
const defaultVerbose = false
const defaultHttpTracing = false

const minTokenLength = 10 // SFx Access Tokens are 22 chars long in 2019 but accept 10 or more chars just in case

const ingestURLEnv = "INGEST"
const tokenEnv = "TOKEN"
const fastIngestEnv = "FAST_INGEST"
const reportingDelayEnv = "REPORTING_RATE"
const reportingTimeoutEnv = "REPORTING_TIMEOUT"
const verboseEnv = "VERBOSE"
const httpTracingEnv = "HTTP_TRACING"

type Configuration struct {
	IngestURL        string
	Token            string
	FastIngest       bool
	ReportingDelay   time.Duration
	ReportingTimeout time.Duration
	Verbose          bool
	HttpTracing      bool
}

func New() Configuration {
	return Configuration{
		IngestURL:        strOrDefault(ingestURLEnv, defaultIngestURL),
		Token:            strOrDefault(tokenEnv, defaultToken),
		FastIngest:       boolOrDefault(fastIngestEnv, defaultFastIngest),
		ReportingDelay:   durationOrDefault(reportingDelayEnv, defaultReportingDuration),
		ReportingTimeout: durationOrDefault(reportingTimeoutEnv, defaultReportingTimeout),
		Verbose:          boolOrDefault(verboseEnv, defaultVerbose),
		HttpTracing:      boolOrDefault(httpTracingEnv, defaultHttpTracing),
	}
}

func (c Configuration) String() string {
	builder := strings.Builder{}
	addLine := func(format string, arg interface{}) { builder.WriteString(fmt.Sprintf(format+"\n", arg)) }

	addLine("Ingest URL        = %v", c.IngestURL)
	addLine("Token             = %v", obfuscatedToken(c.Token))
	addLine("Fast Ingest       = %v", c.FastIngest)
	addLine("Reporting Delay   = %v", c.ReportingDelay.Seconds())
	addLine("Reporting Timeout = %v", c.ReportingTimeout.Seconds())
	addLine("Verbose           = %v", c.Verbose)
	addLine("HTTP Tracing      = %v", c.HttpTracing)

	return builder.String()
}

func obfuscatedToken(token string) string {
	if len(token) < minTokenLength {
		return fmt.Sprintf("<invalid token> minimum %v chars required", minTokenLength)
	}
	return fmt.Sprintf("%s...%s", token[0:2], token[len(token)-2:])
}

func strOrDefault(key, d string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return d
}

func durationOrDefault(key string, d time.Duration) time.Duration {
	str := strOrDefault(key, "nan")

	if seconds, err := strconv.Atoi(str); err == nil {
		return time.Second * time.Duration(seconds)
	}

	log.Printf("can't parse number of seconds: %s\n", str)
	return d
}

func boolOrDefault(key string, d bool) bool {
	str := strOrDefault(key, "")

	if trueOrFalse, err := strconv.ParseBool(str); err == nil {
		return trueOrFalse
	}

	log.Printf("can't parse bool: %s\n", str)
	return d
}
