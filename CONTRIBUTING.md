# Local build

To build and package extension to a zip file:

```
make
```

To deploy the layer as a new version:

```
make add-layer-version PROFILE=integrations REGIONS=us-east-1 LAYER_NAME=signalfx-extension-wrapper CI=t
```

To make the layer globally available (example for us-east-1 only):

```
make add-layer-version-permission PROFILE=integrations REGIONS=us-east-1 LAYER_NAME=signalfx-extension-wrapper CI=t
```

Variables explanation:
* PROFILE - the name of the [AWS CLI profile](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html) - indicates the AWS account where the layer will be published
* REGIONS - a space-delimited list of regions where the layer will be published
* LAYER_NAME - the name of the layer

The published layer can be attached to any AWS Lambda function, regardless of a runtime.


### Deploy to a specified set of regions (example)

```
make all add-layer-version add-layer-version-permission PROFILE=rnd REGIONS="us-east-1 ap-northeast-1" CI=t
```

# CircleCI

### Overview

This project is ready to run with CircleCI environment.
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
        * using SignalFlow client, verify that Splunk Observability backend received all expected datapoints
    * make the published versions publicly available (this step runs only on the `master` branch,
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
