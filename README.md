# OVERVIEW

The SignalFx Lambda Extension Layer provides customers with a simplified runtime-independent interface to collect high-resolution, low-latency metrics on AWS Lambda Function execution. The Extension Layer tracks metrics for cold start, invocation count, function lifetime and termination condition enabling customers to efficiently and effectively monitor their Lambda Functions with minimal overhead.

# METRICS

|Metric name|Type|Description|
|---|---|---|
|lambda.function.invocation|Counter|Number of function calls.|
|lambda.function.initialization|Counter|Number of extension starts. This is the equivalent of the number of cold starts.|
|lambda.function.initialization.latency|Gauge|Time spent between the function execution and its first invocation (in milliseconds).|
|lambda.function.shutdown|Counter|Number of extension shutdowns.|
|lambda.function.lifetime|Gauge|Lifetime of one extension (in milliseconds).| 

Reported dimension:

|Dimension name|Description|
|---|---|
|AWSUniqueId|Unique ID used for correlation with the results of AWS/Lambda tag syncing.|
|aws_arn|ARN of the Lambda function instance|
|aws_region|AWS Region|
|aws_account_id|AWS Account ID|
|aws_function_name|The name of the Lambda function|
|aws_function_version|The version of the Lambda function|
|aws_function_qualifier|AWS Function Version Qualifier (version or version alias, available only for invocations)|
|aws_function_runtime|AWS execution environment|
|cause|It is only present in the shutdown metric. It holds the reason of the shutdown.|

# CONFIGURATION

The main entry point for the configuration is the [`config.Configuration`](internal/config/config.go) struct.
The extension expects to receive configuration parameters via environment variables.

Supported variables:

|Name|Default value|Accepted values|Description|
|---|---|---|---|
|SPLUNK_REALM|`us0`| |The name of your organization's realm as described [here](https://dev.splunk.com/observability/docs/realms_in_endpoints/). It is used to build a standard endpoint for ingesting metrics.|
|SPLUNK_INGEST_URL| |`https://ingest.eu0.signalfx.com/v2/datapoint`|A metrics ingest endpoint as described [here](https://developers.signalfx.com/ingest_data_reference.html#tag/Send-Metrics). It overrides the endpoint defined by the `SPLUNK_REALM` variable and it can be used to point to non standard endpoints.|
|SPLUNK_ACCESS_TOKEN| | |Access token as described [here](https://docs.signalfx.com/en/latest/admin-guide/tokens.html#access-tokens).|
|FAST_INGEST|`true`|`true` or `false`|Determines the strategy used to send data points. `true` for sending metrics on every lambda invocation. With `false` metrics will be buffered and send out on intervals defined by `REPORTING_RATE`.|
|REPORTING_RATE|`15`|Integer (seconds). Minimum value is 1s.|Specifies how often data points are sent to Splunk Observability. The extension is optimized not to report counters of 0, which may cause longer reporting intervals than configured. This variable is used only when the `FAST_INGEST` one is set to `false `.|   
|REPORTING_TIMEOUT|`5`|Integer (seconds). Minimum value is 1s.|Specifies the time to fail datapoint requests if they don't succeed.|
|VERBOSE|`false`|`true` or `false`|Enables verbose logging. Logs are stored in the CloudWatch Log group associated with the Lambda function.|
|HTTP_TRACING|`false`|`true` or `false`|Enables detailed logs on HTTP calls to Splunk Observability.|
