# OVERVIEW

The SignalFx Lambda Extension Layer provides customers with a simplified runtime-independent interface to collect high-resolution, low-latency metrics on Lambda Function execution. The Extension Layer tracks metrics for cold start, invocation count, function lifetime and termination condition enabling customers to efficiently and effectively monitor their Lambda Functions with minimal overhead.

# METRICS

|Metric name|Type|Description|
|---|---|---|
|lambda.function.invocation|Counter|Number of function calls.|
|lambda.function.initialization|Counter|Number of extension starts. This is the equivalent of the number of cold starts.|
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
|aws_function_qualifier|AWS Function Version Qualifier (version or version alias)|
|aws_function_runtime|AWS execution environment|
|cause|It is only present in the shutdown metric. It holds the reason of the shutdown.|

# CONFIGURATION

The main entry point for the configuration is the [`config.Configuration`](internal/config/config.go) struct.
The extension expects to receive configuration parameters via environment variables.

Supported variables:

|Name|Default value|Accepted values|Description|
|---|---|---|---|
|INGEST|`https://ingest.signalfx.com/v2/datapoint`|`https://ingest.{REALM}.signalfx.com/v2/datapoint`|A metrics ingest endpoint as described [here](https://developers.signalfx.com/ingest_data_reference.html#tag/Send-Metrics).|
|TOKEN| | |An access token as described [here](https://docs.signalfx.com/en/latest/admin-guide/tokens.html#access-tokens).|
|REPORTING_RATE|`15`|An integer (seconds). Minimum value is 1s.|Specifies how often data points are sent to SignalFx. It could happen that data points are less dense than expected. A possible reason is that the extension does not report counters of 0 value (due to optimization).|  
|REPORTING_TIMEOUT|`5`|An integer (seconds). Minimum value is 1s.|Specifies the time to fail datapoint requests if they don't succeed.|
|VERBOSE|`false`|`true` or `false`|Enables verbose logging. Logs are stored in a CloudWatch Logs group associated with a Lambda function.|

# BUILDING

To build and package extension to a zip file:

```
make
```

To publish the zip as a layer:

```
make publish PROFILE=integrations REGION=us-east-1 NAME=signalfx-extension-wrapper
```

Variables explanation:
* PROFILE - the name of the [AWS CLI profile](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html) - indicates the AWS account where the layer will be published
* REGION - indicates the region where the layer will be published
* NAME - the name of the layer

The published layer can be attached to any lambda function.
