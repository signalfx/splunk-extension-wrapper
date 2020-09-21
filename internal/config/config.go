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
const defaultReportingDuration = time.Duration(15) * time.Second
const defaultVerbose = false

const minTokenLength = 10 // SFx Access Tokens are 22 chars long in 2019 but accept 10 or more chars just in case

const ingestURLEnv = "INGEST"
const tokenEnv = "TOKEN"
const reportingDelay = "REPORTING_DELAY"
const verbose = "VERBOSE"

type Configuration struct {
	IngestURL      string
	Token          string
	ReportingDelay time.Duration
	Verbose        bool
}

func New() Configuration {
	return Configuration{
		IngestURL:      strOrDefault(ingestURLEnv, defaultIngestURL),
		Token:          strOrDefault(tokenEnv, defaultToken),
		ReportingDelay: durationOrDefault(reportingDelay, defaultReportingDuration),
		Verbose:        boolOrDefault(verbose, defaultVerbose),
	}
}

func (c Configuration) String() string {
	builder := strings.Builder{}
	addLine := func(format string, a ...interface{}) { builder.WriteString(fmt.Sprintf(format+"\n", a...)) }

	addLine("Ingest URL      = %v", c.IngestURL)
	addLine("Token           = %v", obfuscatedToken(c.Token))
	addLine("Reporting Delay = %v", c.ReportingDelay.Seconds())
	addLine("Verbose         = %v", c.Verbose)

	return builder.String()
}

func obfuscatedToken(token string) string {
	if len(token) < minTokenLength {
		return "<invalid token>"
	}
	return fmt.Sprintf("%s...%s", token[0:2], token[len(token)-2:])
}

func strOrDefault(key, d string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	} else {
		return d
	}
}

func durationOrDefault(key string, d time.Duration) time.Duration {
	str := strOrDefault(key, "nan")
	seconds, err := strconv.Atoi(str)

	if err != nil {
		log.Printf("can't parse number of seconds: %s\n", str)
		return d
	}

	return time.Second * time.Duration(seconds)
}

func boolOrDefault(key string, d bool) bool {
	str := strOrDefault(key, "")
	trueOrFalse, err := strconv.ParseBool(str)

	if err != nil {
		return d
	}

	return trueOrFalse
}
