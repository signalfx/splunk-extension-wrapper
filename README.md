# OVERVIEW

The SignalFx Lambda Extension Layer provides customers with a simplified runtime-independent interface to collect high-resolution, low-latency metrics on Lambda Function execution. The Extension Layer tracks metrics for cold start, invocation count, function lifetime and termination condition enabling customers to efficiently and effectively monitor their Lambda Functions with minimal overhead.

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
|INGEST|`https://ingest.signalfx.com/v2/datapoint`|`https://ingest.{REALM}.signalfx.com/v2/datapoint`|A metrics ingest endpoint as described [here](https://developers.signalfx.com/ingest_data_reference.html#tag/Send-Metrics).|
|TOKEN| | |An access token as described [here](https://docs.signalfx.com/en/latest/admin-guide/tokens.html#access-tokens).|
|REPORTING_RATE|`15`|An integer (seconds). Minimum value is 1s.|Specifies how often data points are sent to SignalFx. It could happen that data points are less dense than expected. A possible reason is that the extension does not report counters of 0 value (due to optimization).|  
|REPORTING_TIMEOUT|`5`|An integer (seconds). Minimum value is 1s.|Specifies the time to fail datapoint requests if they don't succeed.|
|VERBOSE|`false`|`true` or `false`|Enables verbose logging. Logs are stored in a CloudWatch Logs group associated with a Lambda function.|
|HTTP_TRACING|`false`|`true` or `false`|Enables detailed logs on HTTP calls to SignalFx.|

# BUILDING

To build and package extension to a zip file:

```
make
```

To deploy the layer as a new version:

```
make add-layer-version PROFILE=integrations REGIONS=us-east-1 LAYER_NAME=signalfx-extension-wrapper
```

To make a layer globally available to all AWS accounts (example for us-east-1 only):

```
make add-layer-version-permission PROFILE=integrations REGIONS=us-east-1 LAYER_NAME=signalfx-extension-wrapper
```

Variables explanation:
* PROFILE - the name of the [AWS CLI profile](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html) - indicates the AWS account where the layer will be published
* REGIONS - a space-delimited list of regions where the layer will be published
* LAYER_NAME - the name of the layer

The published layer can be attached to any lambda function.


### Deploy to a specified set of regions (example)

```
PROFILE=rnd REGIONS="us-east-1 ap-northeast-1" CI=t make all add-layer-version add-layer-version-permission
```

# CircleCI 

### Overview

This project is ready to run win CircleCI environment. 
The CircleCI workflow configured in [config.yml](.circleci/config.yml) consists of:
* continuous integration capabilities (run on every branch)
    * unit test
    * package the extension as a Lambda Layer (an artifact deployable to AWS Lambda)
* continuous delivery capabilities (run on the `master` branch 
  and other branches with the `pipeline-` prefix)
    * publish a new version of the layer to all available regions
    * test the layer in the selected set of regions
        * create a lambda function (with the layer attached)
        * invoke the function a couple of times
        * remove the function
        * verify that all the function invocations were registered in SignalFx
    * make the published versions publicly available (this step is run only on the `master` branch, 
      and it has to be manually approved)
    
### Set up

While the CI part can work right out of the box, the CD part requires a few setup steps. 
This includes:

* Create an AWS user account with at least minimal set of privileges
    * EC2
        * describe regions
    * Lambda
        * publish a layer version
        * add permission to a layer version
        * create a function
        * invoke a function
        * delete a function
    
* Create a basic role for a function
    ```
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "Service": "lambda.amazonaws.com"
          },
          "Action": "sts:AssumeRole"
        }
      ]
    }
    ```

* Environment variables
    * Required
        * AWS_ACCESS_KEY_ID - the id of the access key for the AWS account where the layer will be tested and published
        * AWS_SECRET_ACCESS_KEY - the secret for the access key (the one defined above)
        * AWS_DEFAULT_REGION - actually it doesn't matter which region, but it is required
        * FUNCTION_REALM - realm to which data points will be published (testing)
        * FUNCTION_TOKEN - access token of an organization to which data points will be published (testing)
        * PROFILE - should be set to 'default'
    * Optional
        * REGIONS (space separated list of regions) - controls to which regions 
