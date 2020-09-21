# METRICS

|Metric name|Description|
|---|---|
|lambda.extension.function.invocation|Number of function calls.|
|lambda.extension.environment.initialization|Number of extension starts.|
|lambda.extension.environment.shutdown|Number of extension shutdowns.|
|lambda.extension.environment.duration|Lifetime of one extension (in milliseconds).| 
|lambda.extension.environment.active|It is periodically emitted over lifetime of an extension (value is always 1).| 

Reported dimension:

|Dimension name|Description|
|---|---|
|AWSUniqueId|Our internal ID for AWS. It is here to support tag sync of AWS/Lambda namespace.|
|name|The name of the function|
|version|The version of the function.|
|cause|It is only present in the shutdown metric. It holds the reason of the shutdown.|

# CONFIGURATION

The main entry point for the configuration is the `config.Configuration` struct.
The extension expects to receive configuration parameters via environment variables.

Supported variables:

|Name|Default value|Accepted values|
|---|---|---|
|INGEST|`https://ingest.signalfx.com/v2/datapoint`|`https://ingest.{REALM}.signalfx.com/v2/datapoint`
|TOKEN|| |Access token to the ingest endpoint.
|REPORTING_DELAY|`15`|An integer (seconds)  
|VERBOSE|`false`|`true` or `false`

# BUILDING

To build and package extension to a zip file:

```
make
```

To publish the zip as a layer:

```
make publish PROFILE=integrations REGION=us-east-1 NAME=lambda-extension-wrapper
```

The published layer can be attached to any lambda function.
