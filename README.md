# OVERVIEW

The SignalFx Lambda Extension Layer provides customers with a simplified runtime-independent interface to collect high-resolution, low-latency metrics on Lambda Function execution. The Extension Layer tracks metrics for cold start, invocation count, function lifetime and termination condition enabling customers to efficiently and effectively monitor their Lambda Functions with minimal overhead.

# METRICS

|Metric name|Type|Description|
|---|---|---|
|lambda.extension.function.invocation|Counter|Number of function calls.|
|lambda.extension.environment.initialization|Counter|Number of extension starts.|
|lambda.extension.environment.shutdown|Counter|Number of extension shutdowns.|
|lambda.extension.environment.duration|Gauge|Lifetime of one extension (in milliseconds).| 
|lambda.extension.environment.active|Gauge|It is periodically emitted over lifetime of an extension (value is always 1).| 

Reported dimension:

|Dimension name|Description|
|---|---|
|AWSUniqueId|Unique ID used for correlation with the results of AWS/Lambda tag syncing.|
|name|The name of the function.|
|version|The version of the function.|
|cause|It is only present in the shutdown metric. It holds the reason of the shutdown.|

# CONFIGURATION

The main entry point for the configuration is the [`config.Configuration`](internal/config/config.go) struct.
The extension expects to receive configuration parameters via environment variables.

Supported variables:

|Name|Default value|Accepted values|Description|
|---|---|---|---|
|INGEST|`https://ingest.signalfx.com/v2/datapoint`|`https://ingest.{REALM}.signalfx.com/v2/datapoint`|A metrics ingest endpoint as described [here](https://developers.signalfx.com/ingest_data_reference.html#tag/Send-Metrics).|
|TOKEN| | |An access token as described [here](https://docs.signalfx.com/en/latest/admin-guide/tokens.html#access-tokens).|
|REPORTING_DELAY|`15`|An integer (seconds)|Sets the interval metrics are reported to SignalFx.|  
|VERBOSE|`false`|`true` or `false`|Enables verbose logging.|

# BUILDING

To build and package extension to a zip file:

```
make
```

To publish the zip as a layer:

```
make publish PROFILE=integrations REGION=us-east-1 NAME=lambda-extension-wrapper
```

Variables explanation:
* PROFILE - the name of the [AWS CLI profile](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html) - indicates the AWS account where the layer will be published
* REGION - indicates the region where the layer will be published
* NAME - the name of the layer

The published layer can be attached to any lambda function.
