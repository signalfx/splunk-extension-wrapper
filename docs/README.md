# SignalFx Lambda Extension Layer

The SignalFx Lambda Extension Layer provides customers with a simplified runtime-independent
interface to collect high-resolution, low-latency metrics on Lambda Function execution. The
Extension Layer tracks metrics for cold start, invocation count, function lifetime and termination
condition enabling customers to efficiently and effectively monitor their Lambda Functions with
minimal overhead.

## Concepts

The SignalFx Lambda Extension Layer was designed to send metrics in real-time and moreover to have
minimal impact on a monitored function. To meet these needs, we introduce two ingest
modes: [fast ingest](#Fast-ingest) and [buffering](#Buffering). You have to choose the one that best
suits your case. Please refer to [the configuration section](#Configuration) to check how to switch
between available modes.

### Fast ingest

This mode provides the behaviour that is closest to the real-time, because it sends a metric update
every time a monitored function is invoked. This may have significant impact on overall duration of
a function and consequently it may result in poor user experience. Fortunately this effect can be
eliminated by enabling the Fast Invoke Response in the function, but this may come at the cost of
increased concurrency and longer billed duration of the function.

This mode is best suited for functions that:
* are rarely called
* can accept increased concurrency
* require realm-time metrics

### Buffering

This mode sacrifices real-time characteristic and aims to minimize the impact on a monitored
function. Data points will be buffered internally and sent every interval that has been configured.
Unfortunately, this mode comes with a pitfall that is rooted in the AWS extension architecture.
Namely, the last chunk of buffered data points can be sent with a significant delay, because Lambda
may freeze the execution environment. This happens when each process in the execution environment
has completed and there are no pending events.

This mode is better for users who do not need near real-time feedback and don't want to increase
function latency.

**_Note:_** In general, buffering mode should not be used for functions that are invoked less
frequently than the reporting interval, as such a combination may lead to data points delays greater
than the reporting interval.

## Installation

You can attach the SignalFx Lambda Extension Layer to your Lambda Function as a layer. This can be
done using: AWS CLI, AWS Console, AWS CloudFormation, etc. Please refer to the corresponding
documentation of the approach you use.

**_Note:_** Choose the Layer ARN from the same region as your monitored function.
Check [the newest SignalFx Lambda Extension Layer versions](lambda-extension-versions.md)
for the adequate ARN.

It is important to tell the Extension Layer where to send data points. Use environment variables of
your Lambda Function to configure the Extension Layer.
See [the configuration section](#Configuration) for all configuration options.

Now you should see data points coming to your organization. Go
to [the dedicated dashboard](#Built-in-dashboard) to verify your setup. You can also build your own
dashboard based on [the metrics supported](#Metrics).

If you cannot see data points coming check [the troubleshooting instructions](#TROUBLESHOOTING).

## Built-in dashboard

You can build your own dashboard based on the metrics supported, but we encourage you to take a look
at [built-in dashboards](https://docs.signalfx.com/en/latest/getting-started/built-in-content/built-in-dashboards.html#built-in-dashboards)
first. You can find one dedicated for the SignalFx Lambda Extension Layer. It is available under
the `AWS Lambda` dashboard group. Its name is 'Lambda Extension'. The dashboard demonstrates what
could be achieved with [the metrics the Extension Layer supports](#Metrics) and could be a good
starting point for creating your own dashboard.

Some charts in the dashboard will only populate if you
have [metadata synchronization](https://docs.signalfx.com/en/latest/integrations/amazon-web-services.html#importing-account-metadata-and-custom-tags)
for AWS Lambda namespace enabled. Otherwise, they will remain empty. For example, it applies to
the `Environment Details` chart.

## Metrics

The list of all metrics reported by the SignalFx Lambda Extension Layer:

|Metric name|Type|Description|
|---|---|---|
|lambda.function.invocation|Counter|Number of Lambda Function calls.|
|lambda.function.initialization|Counter|Number of extension starts. This is the equivalent of the number of cold starts.|
|lambda.function.initialization.latency|Gauge|Time spent between a start of the extension and the first lambda invocation (in milliseconds).|
|lambda.function.shutdown|Counter|Number of extension shutdowns.|
|lambda.function.lifetime|Gauge|Lifetime of one extension (in milliseconds). Extension lifetime may span multiple lambda invocations.| 

**_Note:_** We currently do not support a metric that tracks execution time of a function. Please
consider using alternative indicators. The lifetime metric may help with functions that are rarely
called. Another indication may be increased function concurrency that may be the result of longer
execution time.

### Dimensions

The list of all dimensions associated with the metrics reported by the SignalFx Lambda Extension Layer:

|Dimension name|Description|
|---|---|
|AWSUniqueId|Unique ID used for correlation with the results of AWS/Lambda tag syncing.|
|aws_arn|ARN of the Lambda Function instance|
|aws_region|AWS Region|
|aws_account_id|AWS Account ID|
|aws_function_name|The name of the Lambda Function|
|aws_function_version|The version of the Lambda Function|
|aws_function_qualifier|AWS Function Version Qualifier (version or version alias, available only for invocations)|
|aws_function_runtime|AWS Lambda execution environment|
|aws_function_shutdown_cause|It is only present in the shutdown metric. It holds the reason of the shutdown.|

## Configuration

The SignalFx Lambda Extension Layer can be configured by environment variables of a Lambda Function.

Minimal configuration should include `SPLUNK_REALM` (or `SPLUNK_INGEST_URL`)
and `SPLUNK_ACCESS_TOKEN` variables, so the layer can identify the organization to which it should
send data points.

Below is the full list of supported environment variables:
 
|Name|Default value|Accepted values|Description|
|---|---|---|---|
|SPLUNK_REALM|`us0`| |The name of your organization's realm as described [here](https://dev.splunk.com/observability/docs/realms_in_endpoints/). It is used to build a standard endpoint for ingesting metrics.|
|SPLUNK_INGEST_URL| |`https://ingest.eu0.signalfx.com/v2/datapoint`|A metrics ingest endpoint as described [here](https://developers.signalfx.com/ingest_data_reference.html#tag/Send-Metrics). It overrides the endpoint defined by the `SPLUNK_REALM` variable and it can be used to point to non standard endpoints.|
|SPLUNK_ACCESS_TOKEN| | |Access token as described [here](https://docs.signalfx.com/en/latest/admin-guide/tokens.html#access-tokens).|
|REPORTING_RATE|`15`|An integer (seconds). Minimum value is 1s.|Specifies how often data points are sent to SignalFx. Due to the way the AWS Lambda execution environment works metrics may be sent less often.|  
|REPORTING_TIMEOUT|`5`|An integer (seconds). Minimum value is 1s.|Specifies metric send operation timeout.|
|VERBOSE|`false`|`true` or `false`|Enables verbose logging. Logs are stored in a CloudWatch Logs group associated with a Lambda Function.|
|HTTP_TRACING|`false`|`true` or `false`|Enables detailed logs on HTTP calls to SignalFx.|


## Troubleshooting

### I do not see data points coming

1. Check [Cloud Watch metrics](https://docs.aws.amazon.com/lambda/latest/dg/monitoring-metrics.html)
   of your Lambda Function. Make sure the Lambda Function is getting invoked. You can also check if
   errors are reported. Sometimes this indicates an issue with the Extension Layer. You can diagnose
   this by skipping to the 4th point.

2. Make sure `SPLUNK_REALM` (or `SPLUNK_INGEST_URL`) and `SPLUNK_ACCESS_TOKEN` variables are
   correctly configured. Refer to [the configuration section](#Configuration).

3. The Extension Layer working in the buffering mode may send data points with significant delay.
   Refer to [the fast ingest section](#Fast-ingest).

4. Enable verbose logging of the Extension Layer as described
   in [the configuration section](#Configuration).
   Check [Cloud Watch logs](https://docs.aws.amazon.com/lambda/latest/dg/monitoring-cloudwatchlogs.html)
   of your Lambda Function.   

