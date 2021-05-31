# Local build

To build and package extension to a zip file:

```shell
make
```

To deploy the layer as a new version:

```shell
make add-layer-version PROFILE=integrations REGIONS=us-east-1 LAYER_NAME=splunk-lambda-extension CI=t
```

To make the layer globally available (example for us-east-1 only):

```shell
make add-layer-version-permission PROFILE=integrations REGIONS=us-east-1 LAYER_NAME=splunk-lambda-extension CI=t
```

Variables explanation:
* PROFILE - the name of the [AWS CLI profile](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html) - indicates the AWS account where the layer will be published
* REGIONS - a space-delimited list of regions where the layer will be published
* LAYER_NAME - the name of the layer

The published layer can be attached to any AWS Lambda function, regardless of a runtime.


### Deploy to a specified set of regions (example)

```shell
make all add-layer-version add-layer-version-permission PROFILE=rnd REGIONS="us-east-1 ap-northeast-1" CI=t
```

# Local e2e tests

You can run e2e test from your machine. It is steered by an input file ([example](test/local_test_example.txt)).
It'll:
* publish a version of the layer you're currently working on in all regions you need
* create/invoke/delete functions as many times as is described in the input file
* remove the layer version created by the test

Make sure you have the prerequisites:

```shell
make; make prepare-test-function
```

Now you can run a test:

```shell
PROFILE=integrations \ 
SPLUNK_REALM=eu0 \
SPLUNK_ACCESS_TOKEN=*** \
scripts/run-local-test.sh test/local_test_example.txt 
```

Required environment variables:
* `PROFILE` - determines which AWS account the functions will be created under
* `SPLUNK_REALM` (or `SPLUNK_INGEST_URL`), `SPLUNK_ACCESS_TOKEN` - defines the organization where the functions will publish data points

### Input file

The input file is a CSV like file. Meaning of the columns starting from the left:
* function name (doesn't have to be unique)
* region in which the function will be created
* how many times the function should be invoked
* delay between function calls

# CircleCI

### Overview

This project is ready to run with CircleCI environment.
The CircleCI workflow configured in [config.yml](.circleci/config.yml) consists of:
* continuous integration capabilities (run on every branch)
    * unit test
    * package the extension as a Lambda Layer (an artifact deployable to AWS Lambda)
* continuous delivery capabilities (run on the `master` branch;
   branches with the `pipeline-` prefix run e2e tests)
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

* For releasing
    * Create an AWS account which will be used for layer publishing.
      It must have the following permissions:
        ```json
        {
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Effect": "Allow",
                    "Action": "ec2:DescribeRegions",
                    "Resource": "*"
                },
                {
                    "Effect": "Allow",
                    "Action": [
                        "lambda:PublishLayerVersion"
                    ],
                    "Resource": "arn:aws:lambda:*:<account_number>:layer:splunk-lambda-extension"
                },
                {
                    "Effect": "Allow",
                    "Action": [
                        "lambda:AddLayerVersionPermission"
                    ],
                    "Resource": "arn:aws:lambda:*:<account_number>:layer:splunk-lambda-extension:*"
                }
            ]
        }
        ```

    * Note: publishing user and testing user are not separated yet, for now add 
      the above permissions for the testing user that will be set up in the next step.

* End-to-end testing
    * Create a basic role for a function (call it `signalfx-extension-wrapper-testing`)
        ```json
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
      
    * Create an AWS account which will be used for testing. 
      It must have the following permissions:
        ```json
        {
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Effect": "Allow",
                    "Action": [
                        "lambda:CreateFunction",
                        "lambda:InvokeFunction",
                        "lambda:DeleteFunction"
                    ],
                    "Resource": [
                        "arn:aws:lambda:*:<account_number>:function:singalfx-extension-wrapper-test-function",
                        "arn:aws:lambda:*:<account_number>:function:singalfx-extension-wrapper-test-fast-ingest-function"
                    ]
                },
                {
                    "Effect": "Allow",
                    "Action": [
                        "lambda:PublishLayerVersion",
                        "lambda:GetLayerVersion",
                        "lambda:AddLayerVersionPermission",
                        "lambda:DeleteLayerVersion"
                    ],
                    "Resource": [
                        "arn:aws:lambda:*:<account_number>:layer:signalfx-extension-wrapper-test",
                        "arn:aws:lambda:*:<account_number>:layer:signalfx-extension-wrapper-test:*"
                    ]
                },
                {
                    "Effect": "Allow",
                    "Action": [
                        "iam:PassRole"
                    ],
                    "Resource": "arn:aws:iam::<account_number>:role/signalfx-extension-wrapper-testing"
                }
            ]
        }
        ```

    * Set up a CircleCI context called `aws-integrations-lambda-extension-user`.
      Set there the following environment variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, 
      `AWS_DEFAULT_REGION`, so they point to the account setup in the previous step.
      
    * Set up a CircleCI context called `eu0-integrations-ingest`.
      Set there the following environment variables: `SPLUNK_REALM`, `SPLUNK_ACCESS_TOKEN`,
      so they point to a Splunk ingest.

* Environment variables
    * Required
        * `AWS_ACCESS_KEY_ID` - the id of the access key for the AWS account where the layer will be tested and published
        * `AWS_SECRET_ACCESS_KEY` - the secret for the access key (the one defined above)
        * `AWS_DEFAULT_REGION` - actually it doesn't matter which region, but it is required
        * `SPLUNK_REALM` - realm to which data points will be published (testing)
        * `SPLUNK_ACCESS_TOKEN` - access token of an organization to which data points will be published (testing)
        * `PROFILE` - should be set to 'default'


# Release

After a change is merged to the `master` branch and e2e test will succeed, you'll get a chance to confirm or cancel a job that publishes the artifacts.
If you decide to proceed with the release, go to the appropriate workflow and approve the `confirm_making_public` job.
This will publish a new version of the layer in each region and will grant every AWS account access to each of the newly published versions.

You can also cancel a workflow instead of approving the `confirm_making_public` job, so not every commit merged to the `master` branch has to be released.

The CD pipeline doesn't publish arns of the new versions to [the version repo](https://github.com/signalfx/lambda-layer-versions/tree/master/lambda-extension),
so this step must be done manually.
You can find the list of the newly published versions in the `publish_layer_versions` job in CricleCI (look for the `bin/versions` file in the `ARTIFACTS` tab of the job).
